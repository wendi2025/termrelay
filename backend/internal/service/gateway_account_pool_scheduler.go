package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const (
	defaultGatewayAccountPoolTopK = 3
	gatewayAccountPoolEnabledEnv  = "SUB2API_ACCOUNT_POOL_SCHEDULER"
	gatewayAccountPoolTopKEnv     = "SUB2API_ACCOUNT_POOL_TOP_K"
)

// Keep the migration runtime outside GatewayService until the selection path is
// switched on. Account IDs are process-global and GatewayService is effectively
// singleton in production, so this preserves health feedback without forcing a
// constructor signature change across unrelated workstreams.
var gatewayAccountPoolRuntimes sync.Map // map[*GatewayService]*AccountPoolScheduler

// AccountPoolSchedulerEnabled is the rollout gate used by the eventual Layer 2
// integration. It defaults off, so merging the scheduler code cannot change
// production routing until explicitly enabled.
func (s *GatewayService) AccountPoolSchedulerEnabled() bool {
	if s == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(gatewayAccountPoolEnabledEnv))) {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

func gatewayAccountPoolTopK() int {
	value := strings.TrimSpace(os.Getenv(gatewayAccountPoolTopKEnv))
	if value == "" {
		return defaultGatewayAccountPoolTopK
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultGatewayAccountPoolTopK
	}
	if parsed > 32 {
		return 32
	}
	return parsed
}

func (s *GatewayService) accountPoolScheduler() *AccountPoolScheduler {
	if s == nil {
		return nil
	}
	if value, ok := gatewayAccountPoolRuntimes.Load(s); ok {
		if scheduler, _ := value.(*AccountPoolScheduler); scheduler != nil {
			return scheduler
		}
	}
	scheduler := NewAccountPoolScheduler(gatewayAccountPoolTopK(), AccountPoolScoreWeights{})
	actual, _ := gatewayAccountPoolRuntimes.LoadOrStore(s, scheduler)
	resolved, _ := actual.(*AccountPoolScheduler)
	return resolved
}

func accountPoolSelectionSeed(ctx context.Context, sessionHash string) string {
	if ctx != nil {
		for _, key := range []ctxkey.Key{ctxkey.RequestID, ctxkey.ClientRequestID} {
			if value, ok := ctx.Value(key).(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	if strings.TrimSpace(sessionHash) != "" {
		return strings.TrimSpace(sessionHash)
	}
	// Request IDs are normally present. The time seed is only a defensive
	// fallback for tests/internal callers and prevents all anonymous requests
	// from pinning to the same rendezvous winner.
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// tryAcquireByAccountPoolScheduler is the narrow integration point for Layer 2.
// Callers must perform the existing hard filters first (enabled/schedulable,
// platform/model, quota, RPM, persistent cooldown). This function only ranks
// the remaining candidates, respects the fast circuit breaker, and atomically
// claims the existing Redis-backed concurrency slots.
func (s *GatewayService) tryAcquireByAccountPoolScheduler(
	ctx context.Context,
	groupID *int64,
	sessionHash string,
	available []accountWithLoad,
) (*AccountSelectionResult, bool, error) {
	if s == nil || len(available) == 0 {
		return nil, false, nil
	}
	scheduler := s.accountPoolScheduler()
	if scheduler == nil {
		return nil, false, nil
	}

	stickyAccountID := int64(0)
	if sessionHash != "" && s.cache != nil {
		stickyAccountID, _ = s.cache.GetSessionAccountID(ctx, derefGroupID(groupID), sessionHash)
	}

	inputs := make([]accountPoolCandidateInput, 0, len(available))
	for _, item := range available {
		if item.account == nil {
			continue
		}
		inputs = append(inputs, accountPoolCandidateInput{
			Account: item.account,
			Load:    item.loadInfo,
			Sticky:  stickyAccountID > 0 && item.account.ID == stickyAccountID,
			ResetAt: item.account.SessionWindowEnd,
		})
	}
	if len(inputs) == 0 {
		return nil, false, nil
	}

	now := time.Now()
	ranked := scheduler.Rank(inputs, accountPoolSelectionSeed(ctx, sessionHash), now)
	for _, candidate := range ranked {
		account := candidate.Account
		if account == nil || !scheduler.Allow(account.ID, now) {
			continue
		}

		result, err := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
		if err != nil || result == nil || !result.Acquired {
			scheduler.Abandon(account.ID)
			continue
		}
		if !s.checkAndRegisterSession(ctx, account, sessionHash) {
			result.ReleaseFunc()
			scheduler.Abandon(account.ID)
			continue
		}
		if sessionHash != "" && s.cache != nil {
			_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionHash, account.ID, stickySessionTTL)
		}
		selection, selectErr := s.newSelectionResult(ctx, account, true, result.ReleaseFunc, nil)
		if selectErr != nil {
			result.ReleaseFunc()
			scheduler.Abandon(account.ID)
			return nil, true, selectErr
		}
		return selection, true, nil
	}

	// Waiting does not constitute an upstream attempt and therefore must never
	// reserve a half-open probe. Only fully closed circuits are eligible here.
	cfg := s.schedulingConfig()
	for _, candidate := range ranked {
		account := candidate.Account
		if account == nil || scheduler.Health(account.ID, now).CircuitState != "closed" {
			continue
		}
		if !s.checkAndRegisterSession(ctx, account, sessionHash) {
			continue
		}
		selection, selectErr := s.newSelectionResult(ctx, account, false, nil, &AccountWaitPlan{
			AccountID:      account.ID,
			MaxConcurrency: account.Concurrency,
			Timeout:        cfg.FallbackWaitTimeout,
			MaxWaiting:     cfg.FallbackMaxWaiting,
		})
		if selectErr != nil {
			return nil, true, selectErr
		}
		return selection, true, nil
	}
	return nil, false, nil
}

// ReportAccountPoolAttempt feeds only account-attributable upstream outcomes
// into the scheduler. Client cancellation, request validation errors and
// provider-wide credential failures are deliberately excluded so health scores
// cannot be poisoned by events that changing accounts would not fix.
func (s *GatewayService) ReportAccountPoolAttempt(accountID int64, result *ForwardResult, err error) {
	var firstTokenMs *int
	if result != nil {
		firstTokenMs = result.FirstTokenMs
	}
	s.ReportAccountPoolOutcome(accountID, firstTokenMs, err)
}

// BeginAccountPoolAttempt reserves the fast-breaker half-open probe immediately
// before a real upstream attempt. Callers using the generic Gateway scheduler
// already do this during selection; provider-specific OpenAI/Grok paths call it
// after acquiring their existing Redis concurrency slot.
func (s *GatewayService) BeginAccountPoolAttempt(accountID int64) bool {
	if s == nil || accountID <= 0 || !s.AccountPoolSchedulerEnabled() {
		return true
	}
	scheduler := s.accountPoolScheduler()
	if scheduler == nil {
		return true
	}
	return scheduler.Allow(accountID, time.Now())
}

// AbandonAccountPoolAttempt releases a half-open reservation when no
// account-attributable upstream outcome was observed (for example client
// cancellation before the probe completes).
func (s *GatewayService) AbandonAccountPoolAttempt(accountID int64) {
	if s == nil || accountID <= 0 || !s.AccountPoolSchedulerEnabled() {
		return
	}
	if scheduler := s.accountPoolScheduler(); scheduler != nil {
		scheduler.Abandon(accountID)
	}
}

// ReportAccountPoolOutcome is the provider-neutral feedback boundary for
// request paths that do not execute GatewayService.Forward directly (OpenAI,
// Gemini native/compat, Grok and Antigravity adapters). Keeping classification
// here guarantees every provider uses the same 429/5xx/transport/cancel rules.
func (s *GatewayService) ReportAccountPoolOutcome(accountID int64, firstTokenMs *int, err error) {
	if s == nil || accountID <= 0 || !s.AccountPoolSchedulerEnabled() {
		return
	}
	scheduler := s.accountPoolScheduler()
	if scheduler == nil {
		return
	}
	report, success, transient := classifyAccountPoolAttempt(err)
	if !report {
		// Do not poison health for client/request-scoped failures, but release
		// any reserved half-open probe so a later request can test the account.
		scheduler.Abandon(accountID)
		return
	}
	scheduler.Report(accountID, success, transient, firstTokenMs, time.Now())
}

func classifyAccountPoolAttempt(err error) (report, success, transient bool) {
	if err == nil {
		return true, true, false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false, false, false
	}

	var failoverErr *UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		if !failoverErr.ShouldReportAccountScheduleFailure() {
			return false, false, false
		}
		status := failoverErr.StatusCode
		// 429 already owns a persistent account/model cooldown. Feeding it into
		// the fast transient breaker would double-penalize the same signal.
		if status == http.StatusTooManyRequests {
			return true, false, false
		}
		if status == http.StatusRequestTimeout || status == http.StatusBadGateway ||
			status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout ||
			status == 529 || status >= 500 {
			return true, false, true
		}
		return true, false, false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true, false, true
	}
	// Several existing forwarders intentionally sanitize transport errors before
	// returning them, which removes the original net.Error cause. The stable
	// prefix is only emitted after an upstream transport attempt, so preserve
	// that signal for breaker health without weakening the sanitized client text.
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(err.Error())), "upstream request failed:") {
		return true, false, true
	}
	return false, false, false
}
