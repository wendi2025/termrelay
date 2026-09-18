package service

import (
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// AccountPoolScoreWeights controls the provider-neutral account pool score.
// Positive weights always mean "prefer the healthier/better candidate".
// The values are intentionally independent from provider protocol concerns so
// OpenAI, Anthropic, Gemini, Grok and generic API-key pools can share one core.
type AccountPoolScoreWeights struct {
	Priority      float64
	Load          float64
	Queue         float64
	ErrorRate     float64
	TTFT          float64
	Reset         float64
	QuotaHeadroom float64
	Sticky        float64
}

func defaultAccountPoolScoreWeights() AccountPoolScoreWeights {
	return AccountPoolScoreWeights{
		Priority:      1.00,
		Load:          1.00,
		Queue:         0.45,
		ErrorRate:     1.20,
		TTFT:          0.55,
		Reset:         0.35,
		QuotaHeadroom: 0.50,
		Sticky:        0.60,
	}
}

// accountPoolRuntimeStats contains request-result feedback that is safe to
// update on the gateway hot path. It deliberately stores no request content or
// credentials.
type accountPoolRuntimeStats struct {
	accounts sync.Map // map[int64]*accountPoolRuntimeStat
	count    atomic.Int64
}

type accountPoolRuntimeStat struct {
	errorRateBits atomic.Uint64
	ttftBits      atomic.Uint64
	samples       atomic.Int64
	lastUnixNano  atomic.Int64
}

func newAccountPoolRuntimeStats() *accountPoolRuntimeStats {
	return &accountPoolRuntimeStats{}
}

func (s *accountPoolRuntimeStats) loadOrCreate(accountID int64) *accountPoolRuntimeStat {
	if s == nil || accountID <= 0 {
		return nil
	}
	if value, ok := s.accounts.Load(accountID); ok {
		if stat, _ := value.(*accountPoolRuntimeStat); stat != nil {
			return stat
		}
	}
	stat := &accountPoolRuntimeStat{}
	stat.errorRateBits.Store(math.Float64bits(0))
	stat.ttftBits.Store(math.Float64bits(math.NaN()))
	actual, loaded := s.accounts.LoadOrStore(accountID, stat)
	if !loaded {
		s.count.Add(1)
		return stat
	}
	existing, _ := actual.(*accountPoolRuntimeStat)
	return existing
}

func updateAccountPoolEWMA(target *atomic.Uint64, sample, alpha float64, nanIsUnset bool) {
	for {
		oldBits := target.Load()
		old := math.Float64frombits(oldBits)
		var next float64
		if nanIsUnset && math.IsNaN(old) {
			next = sample
		} else {
			next = alpha*sample + (1-alpha)*old
		}
		if target.CompareAndSwap(oldBits, math.Float64bits(next)) {
			return
		}
	}
}

// report records one actual upstream attempt. The 0.2 EWMA reacts quickly to
// a bad account without making a single transient error dominate for long.
func (s *accountPoolRuntimeStats) report(accountID int64, success bool, firstTokenMs *int) {
	stat := s.loadOrCreate(accountID)
	if stat == nil {
		return
	}
	errorSample := 1.0
	if success {
		errorSample = 0
	}
	updateAccountPoolEWMA(&stat.errorRateBits, errorSample, 0.2, false)
	if firstTokenMs != nil && *firstTokenMs > 0 {
		updateAccountPoolEWMA(&stat.ttftBits, float64(*firstTokenMs), 0.2, true)
	}
	stat.samples.Add(1)
	stat.lastUnixNano.Store(time.Now().UnixNano())
}

type accountPoolFeedback struct {
	ErrorRate    float64
	TTFTMs       float64
	HasTTFT      bool
	Samples      int64
	LastObserved time.Time
}

func (s *accountPoolRuntimeStats) snapshot(accountID int64) accountPoolFeedback {
	if s == nil || accountID <= 0 {
		return accountPoolFeedback{}
	}
	value, ok := s.accounts.Load(accountID)
	if !ok {
		return accountPoolFeedback{}
	}
	stat, _ := value.(*accountPoolRuntimeStat)
	if stat == nil {
		return accountPoolFeedback{}
	}
	out := accountPoolFeedback{
		ErrorRate: clampAccountPool01(math.Float64frombits(stat.errorRateBits.Load())),
		Samples:   stat.samples.Load(),
	}
	if v := stat.lastUnixNano.Load(); v > 0 {
		out.LastObserved = time.Unix(0, v).UTC()
	}
	if ttft := math.Float64frombits(stat.ttftBits.Load()); !math.IsNaN(ttft) {
		out.TTFTMs = ttft
		out.HasTTFT = true
	}
	return out
}

// accountPoolCircuitBreaker is an in-process fast path. Persistent provider
// rate-limit windows remain authoritative; this breaker only prevents a burst
// of transient failures from repeatedly selecting the same account while the
// persistent side effects catch up.
type accountPoolCircuitBreaker struct {
	states sync.Map // map[int64]*accountPoolCircuitState
}

type accountPoolCircuitState struct {
	mu                  sync.Mutex
	consecutiveFailures int
	openUntil           time.Time
	halfOpenInFlight    bool
}

func (b *accountPoolCircuitBreaker) state(accountID int64) *accountPoolCircuitState {
	if b == nil || accountID <= 0 {
		return nil
	}
	actual, _ := b.states.LoadOrStore(accountID, &accountPoolCircuitState{})
	state, _ := actual.(*accountPoolCircuitState)
	return state
}

// allow implements closed/open/half-open. Once cooldown expires, exactly one
// request is admitted as the half-open probe; concurrent requests keep using
// other accounts until that probe reports a result.
func (b *accountPoolCircuitBreaker) allow(accountID int64, now time.Time) bool {
	state := b.state(accountID)
	if state == nil {
		return true
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.openUntil.IsZero() {
		return true
	}
	if now.Before(state.openUntil) {
		return false
	}
	if state.halfOpenInFlight {
		return false
	}
	state.halfOpenInFlight = true
	return true
}

// report closes a successful circuit immediately. Repeated transient failures
// use exponential cooldown capped at one minute. Non-transient failures do not
// trip this fast breaker because permanent/auth/rate-limit handling belongs to
// the existing account/rate-limit state machine.
func (b *accountPoolCircuitBreaker) report(accountID int64, success, transient bool, now time.Time) {
	state := b.state(accountID)
	if state == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	state.halfOpenInFlight = false
	if success {
		state.consecutiveFailures = 0
		state.openUntil = time.Time{}
		return
	}
	if !transient {
		return
	}
	state.consecutiveFailures++
	if state.consecutiveFailures < 2 {
		return
	}
	exponent := state.consecutiveFailures - 2
	if exponent > 5 {
		exponent = 5
	}
	cooldown := time.Second * time.Duration(1<<exponent)
	if cooldown > time.Minute {
		cooldown = time.Minute
	}
	state.openUntil = now.Add(cooldown)
}

type accountPoolCandidateInput struct {
	Account       *Account
	Load          *AccountLoadInfo
	Sticky        bool
	ResetAt       *time.Time
	QuotaHeadroom *float64
}

type accountPoolCandidateScore struct {
	Account *Account
	Score   float64

	PriorityFactor      float64
	LoadFactor          float64
	QueueFactor         float64
	ErrorFactor         float64
	TTFTFactor          float64
	ResetFactor         float64
	QuotaHeadroomFactor float64
	StickyFactor        float64
}

// scoreAccountPoolCandidates scores only candidates that already passed hard
// eligibility checks. Missing runtime/load signals use neutral values instead
// of pretending that missing data means zero load or perfect health.
func scoreAccountPoolCandidates(inputs []accountPoolCandidateInput, stats *accountPoolRuntimeStats, weights AccountPoolScoreWeights, now time.Time) []accountPoolCandidateScore {
	if len(inputs) == 0 {
		return nil
	}
	minPriority, maxPriority := math.MaxInt, math.MinInt
	maxWaiting := 0
	minTTFT, maxTTFT := math.MaxFloat64, 0.0
	hasTTFT := false
	minReset, maxReset := math.MaxFloat64, 0.0
	hasReset := false

	feedback := make(map[int64]accountPoolFeedback, len(inputs))
	for _, input := range inputs {
		if input.Account == nil {
			continue
		}
		priority := input.Account.Priority
		if priority < minPriority {
			minPriority = priority
		}
		if priority > maxPriority {
			maxPriority = priority
		}
		if input.Load != nil && input.Load.WaitingCount > maxWaiting {
			maxWaiting = input.Load.WaitingCount
		}
		fb := stats.snapshot(input.Account.ID)
		feedback[input.Account.ID] = fb
		if fb.HasTTFT {
			hasTTFT = true
			if fb.TTFTMs < minTTFT {
				minTTFT = fb.TTFTMs
			}
			if fb.TTFTMs > maxTTFT {
				maxTTFT = fb.TTFTMs
			}
		}
		if input.ResetAt != nil && input.ResetAt.After(now) {
			remaining := input.ResetAt.Sub(now).Seconds()
			hasReset = true
			if remaining < minReset {
				minReset = remaining
			}
			if remaining > maxReset {
				maxReset = remaining
			}
		}
	}

	out := make([]accountPoolCandidateScore, 0, len(inputs))
	for _, input := range inputs {
		account := input.Account
		if account == nil {
			continue
		}
		priorityFactor := 1.0
		if maxPriority > minPriority {
			priorityFactor = 1 - float64(account.Priority-minPriority)/float64(maxPriority-minPriority)
		}
		loadFactor, queueFactor := 0.5, 0.5
		if input.Load != nil {
			loadFactor = 1 - clampAccountPool01(float64(input.Load.LoadRate)/100)
			if maxWaiting > 0 {
				queueFactor = 1 - clampAccountPool01(float64(input.Load.WaitingCount)/float64(maxWaiting))
			} else {
				queueFactor = 1
			}
		}
		fb := feedback[account.ID]
		errorFactor := 1 - clampAccountPool01(fb.ErrorRate)
		ttftFactor := 0.5
		if hasTTFT && fb.HasTTFT {
			if maxTTFT > minTTFT {
				ttftFactor = 1 - clampAccountPool01((fb.TTFTMs-minTTFT)/(maxTTFT-minTTFT))
			} else {
				ttftFactor = 1
			}
		}
		resetFactor := 0.5
		if hasReset && input.ResetAt != nil && input.ResetAt.After(now) {
			remaining := input.ResetAt.Sub(now).Seconds()
			if maxReset > minReset {
				resetFactor = 1 - clampAccountPool01((remaining-minReset)/(maxReset-minReset))
			} else {
				resetFactor = 1
			}
		}
		quotaFactor := 0.5
		if input.QuotaHeadroom != nil {
			quotaFactor = clampAccountPool01(*input.QuotaHeadroom)
		}
		stickyFactor := 0.0
		if input.Sticky {
			stickyFactor = 1
		}

		score := weights.Priority*priorityFactor +
			weights.Load*loadFactor +
			weights.Queue*queueFactor +
			weights.ErrorRate*errorFactor +
			weights.TTFT*ttftFactor +
			weights.Reset*resetFactor +
			weights.QuotaHeadroom*quotaFactor +
			weights.Sticky*stickyFactor
		out = append(out, accountPoolCandidateScore{
			Account:             account,
			Score:               score,
			PriorityFactor:      priorityFactor,
			LoadFactor:          loadFactor,
			QueueFactor:         queueFactor,
			ErrorFactor:         errorFactor,
			TTFTFactor:          ttftFactor,
			ResetFactor:         resetFactor,
			QuotaHeadroomFactor: quotaFactor,
			StickyFactor:        stickyFactor,
		})
	}
	return out
}

func rankAccountPoolCandidates(candidates []accountPoolCandidateScore, topK int, seed string) []accountPoolCandidateScore {
	if len(candidates) == 0 {
		return nil
	}
	ranked := append([]accountPoolCandidateScore(nil), candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score > ranked[j].Score
		}
		if ranked[i].Account.Priority != ranked[j].Account.Priority {
			return ranked[i].Account.Priority < ranked[j].Account.Priority
		}
		return ranked[i].Account.ID < ranked[j].Account.ID
	})
	if topK <= 0 || topK > len(ranked) {
		topK = len(ranked)
	}
	// Deterministic weighted rendezvous inside Top-K avoids a global RNG lock,
	// spreads requests, and remains stable for the same request/session seed.
	top := append([]accountPoolCandidateScore(nil), ranked[:topK]...)
	sort.SliceStable(top, func(i, j int) bool {
		return accountPoolRendezvous(seed, top[i]) > accountPoolRendezvous(seed, top[j])
	})
	return append(top, ranked[topK:]...)
}

func accountPoolRendezvous(seed string, candidate accountPoolCandidateScore) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.FormatInt(candidate.Account.ID, 10)))
	u := float64(h.Sum64()+1) / float64(math.MaxUint64)
	if u <= 0 {
		u = math.SmallestNonzeroFloat64
	}
	weight := candidate.Score
	if math.IsNaN(weight) || math.IsInf(weight, 0) || weight <= 0 {
		weight = 1e-9
	}
	// Weighted rendezvous key. Higher is better.
	return weight / -math.Log(u)
}

func clampAccountPool01(v float64) float64 {
	if math.IsNaN(v) {
		return 0.5
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
