package service

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	accountPoolWeightExtraKey      = "scheduling_weight"
	maxAccountPoolSchedulingWeight = 1000.0
)

// SchedulingWeight returns the operator-configured traffic share for an
// account. It is intentionally stored in Account.Extra during the migration
// phase so the scheduler can support weighted pools without requiring an ORM
// regeneration or a blocking schema rollout. A first-class column can replace
// this accessor later without changing the scheduler contract.
func (a *Account) SchedulingWeight() float64 {
	if a == nil || a.Extra == nil {
		return 1
	}
	weight, ok := accountPoolNumericValue(a.Extra[accountPoolWeightExtraKey])
	if !ok || math.IsNaN(weight) || math.IsInf(weight, 0) || weight <= 0 {
		return 1
	}
	// Keep one malformed or extreme admin value from monopolizing the pool.
	if weight > maxAccountPoolSchedulingWeight {
		return maxAccountPoolSchedulingWeight
	}
	return weight
}

// NormalizeAccountPoolSchedulingWeight validates and normalizes the optional
// operator-facing account traffic weight. An explicit typed value wins over the
// legacy Extra value. The returned map is cloned when the key is present so
// callers never mutate a request map owned by another layer.
func NormalizeAccountPoolSchedulingWeight(extra map[string]any, explicit *float64) (map[string]any, error) {
	raw, hasWeight := any(nil), false
	if extra != nil {
		raw, hasWeight = extra[accountPoolWeightExtraKey]
	}
	if explicit != nil {
		raw = *explicit
		hasWeight = true
	}
	if !hasWeight {
		return extra, nil
	}

	weight, ok := accountPoolNumericValue(raw)
	if !ok || math.IsNaN(weight) || math.IsInf(weight, 0) || weight <= 0 || weight > maxAccountPoolSchedulingWeight {
		return nil, infraerrors.BadRequest(
			"INVALID_ACCOUNT_SCHEDULING_WEIGHT",
			"scheduling_weight must be greater than 0 and at most 1000",
		)
	}

	normalized := make(map[string]any, len(extra)+1)
	for key, value := range extra {
		normalized[key] = value
	}
	normalized[accountPoolWeightExtraKey] = weight
	return normalized, nil
}

func accountPoolNumericValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

type accountPoolHealthTracker struct {
	accounts sync.Map // map[int64]*accountPoolHealthState
}

type accountPoolHealthState struct {
	mu                  sync.Mutex
	consecutiveFailures int
	lastSuccessAt       time.Time
	lastFailureAt       time.Time
}

func (t *accountPoolHealthTracker) state(accountID int64) *accountPoolHealthState {
	if t == nil || accountID <= 0 {
		return nil
	}
	actual, _ := t.accounts.LoadOrStore(accountID, &accountPoolHealthState{})
	state, _ := actual.(*accountPoolHealthState)
	return state
}

func (t *accountPoolHealthTracker) report(accountID int64, success bool, now time.Time) {
	state := t.state(accountID)
	if state == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if success {
		state.consecutiveFailures = 0
		state.lastSuccessAt = now
		return
	}
	state.consecutiveFailures++
	state.lastFailureAt = now
}

type accountPoolHealthStateSnapshot struct {
	ConsecutiveFailures int
	LastSuccessAt       *time.Time
	LastFailureAt       *time.Time
}

func (t *accountPoolHealthTracker) snapshot(accountID int64) accountPoolHealthStateSnapshot {
	if t == nil || accountID <= 0 {
		return accountPoolHealthStateSnapshot{}
	}
	value, ok := t.accounts.Load(accountID)
	if !ok {
		return accountPoolHealthStateSnapshot{}
	}
	state, _ := value.(*accountPoolHealthState)
	if state == nil {
		return accountPoolHealthStateSnapshot{}
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	out := accountPoolHealthStateSnapshot{ConsecutiveFailures: state.consecutiveFailures}
	if !state.lastSuccessAt.IsZero() {
		value := state.lastSuccessAt
		out.LastSuccessAt = &value
	}
	if !state.lastFailureAt.IsZero() {
		value := state.lastFailureAt
		out.LastFailureAt = &value
	}
	return out
}

type accountPoolCircuitSnapshot struct {
	State         string
	CooldownUntil *time.Time
}

func (b *accountPoolCircuitBreaker) snapshot(accountID int64, now time.Time) accountPoolCircuitSnapshot {
	if b == nil || accountID <= 0 {
		return accountPoolCircuitSnapshot{State: "closed"}
	}
	value, ok := b.states.Load(accountID)
	if !ok {
		return accountPoolCircuitSnapshot{State: "closed"}
	}
	state, _ := value.(*accountPoolCircuitState)
	if state == nil {
		return accountPoolCircuitSnapshot{State: "closed"}
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.openUntil.IsZero() {
		return accountPoolCircuitSnapshot{State: "closed"}
	}
	until := state.openUntil
	if now.Before(state.openUntil) {
		return accountPoolCircuitSnapshot{State: "open", CooldownUntil: &until}
	}
	if state.halfOpenInFlight {
		return accountPoolCircuitSnapshot{State: "half_open", CooldownUntil: &until}
	}
	return accountPoolCircuitSnapshot{State: "probe_ready", CooldownUntil: &until}
}

// AccountPoolHealthSnapshot is safe to expose through an admin diagnostics API.
// It contains no request payloads or credentials.
type AccountPoolHealthSnapshot struct {
	ErrorRate           float64
	TTFTMs              float64
	HasTTFT             bool
	Samples             int64
	LastObservedAt      *time.Time
	ConsecutiveFailures int
	LastSuccessAt       *time.Time
	LastFailureAt       *time.Time
	CircuitState        string
	CooldownUntil       *time.Time
}
