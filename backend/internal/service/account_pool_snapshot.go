package service

import (
	"context"
	"time"
)

// AccountPoolRuntimeSnapshot is an operator-safe view of one upstream account.
// It intentionally excludes credentials, request payloads and raw upstream
// errors so it can later back admin diagnostics without creating a secret leak.
type AccountPoolRuntimeSnapshot struct {
	AccountID          int64
	Provider           AccountPoolProviderIdentity
	Enabled            bool
	Schedulable        bool
	Priority           int
	Weight             float64
	MaxConcurrency     int
	CurrentConcurrency int
	WaitingCount       int
	LoadRate           int
	LoadKnown          bool
	HealthStatus       string
	Health             AccountPoolHealthSnapshot
}

// AccountPoolSnapshot returns scheduler/account state plus the authoritative
// Redis-backed concurrency snapshot when available. Missing load data is marked
// unknown rather than being reported as zero load.
func (s *GatewayService) AccountPoolSnapshot(ctx context.Context, account *Account) AccountPoolRuntimeSnapshot {
	if account == nil {
		return AccountPoolRuntimeSnapshot{}
	}

	now := time.Now()
	health := AccountPoolHealthSnapshot{CircuitState: "closed"}
	if s != nil {
		if scheduler := s.accountPoolScheduler(); scheduler != nil {
			health = scheduler.Health(account.ID, now)
		}
	}

	out := AccountPoolRuntimeSnapshot{
		AccountID:      account.ID,
		Provider:       account.ProviderIdentity(),
		Enabled:        account.IsActive(),
		Schedulable:    account.IsSchedulable(),
		Priority:       account.Priority,
		Weight:         account.SchedulingWeight(),
		MaxConcurrency: account.Concurrency,
		HealthStatus:   accountPoolHealthStatus(health),
		Health:         health,
	}

	if s == nil || s.concurrencyService == nil || account.ID <= 0 {
		return out
	}
	loads, err := s.concurrencyService.GetAccountsLoadBatch(ctx, []AccountWithConcurrency{{
		ID:             account.ID,
		MaxConcurrency: account.EffectiveLoadFactor(),
	}})
	if err != nil {
		return out
	}
	load := loads[account.ID]
	if load == nil {
		return out
	}
	out.CurrentConcurrency = load.CurrentConcurrency
	out.WaitingCount = load.WaitingCount
	out.LoadRate = load.LoadRate
	out.LoadKnown = true
	return out
}

func accountPoolHealthStatus(health AccountPoolHealthSnapshot) string {
	switch health.CircuitState {
	case "open":
		return "cooling"
	case "half_open", "probe_ready":
		return "probing"
	}
	if health.Samples == 0 {
		return "unknown"
	}
	if health.ConsecutiveFailures >= 3 || health.ErrorRate >= 0.50 {
		return "unhealthy"
	}
	if health.ConsecutiveFailures > 0 || health.ErrorRate >= 0.20 {
		return "degraded"
	}
	return "healthy"
}
