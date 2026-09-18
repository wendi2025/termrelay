package service

import (
	"math"
	"testing"
	"time"
)

func TestAccountPoolRuntimeStatsEWMA(t *testing.T) {
	stats := newAccountPoolRuntimeStats()
	first := 100
	stats.report(10, true, &first)

	got := stats.snapshot(10)
	if got.ErrorRate != 0 {
		t.Fatalf("expected zero error rate after success, got %v", got.ErrorRate)
	}
	if !got.HasTTFT || got.TTFTMs != 100 {
		t.Fatalf("expected first TTFT sample to initialize EWMA, got %+v", got)
	}
	if got.Samples != 1 || got.LastObserved.IsZero() {
		t.Fatalf("expected sample metadata, got %+v", got)
	}

	second := 300
	stats.report(10, false, &second)
	got = stats.snapshot(10)
	if math.Abs(got.ErrorRate-0.2) > 1e-9 {
		t.Fatalf("expected error EWMA 0.2, got %v", got.ErrorRate)
	}
	if math.Abs(got.TTFTMs-140) > 1e-9 {
		t.Fatalf("expected TTFT EWMA 140ms, got %v", got.TTFTMs)
	}
}

func TestAccountPoolCircuitBreakerHalfOpenProbe(t *testing.T) {
	breaker := &accountPoolCircuitBreaker{}
	now := time.Unix(1000, 0)

	breaker.report(7, false, true, now)
	if !breaker.allow(7, now) {
		t.Fatal("one transient failure must not open the circuit")
	}
	breaker.report(7, false, true, now)

	if breaker.allow(7, now.Add(500*time.Millisecond)) {
		t.Fatal("two consecutive transient failures should open the circuit")
	}
	if !breaker.allow(7, now.Add(2*time.Second)) {
		t.Fatal("first request after cooldown should become the half-open probe")
	}
	if breaker.allow(7, now.Add(2*time.Second)) {
		t.Fatal("only one half-open probe may run at a time")
	}

	breaker.report(7, true, true, now.Add(2*time.Second))
	if !breaker.allow(7, now.Add(2*time.Second)) {
		t.Fatal("successful probe should close the circuit")
	}
}

func TestAccountPoolCircuitBreakerNonTransientFailureDoesNotTrip(t *testing.T) {
	breaker := &accountPoolCircuitBreaker{}
	now := time.Unix(2000, 0)
	for i := 0; i < 5; i++ {
		breaker.report(9, false, false, now)
	}
	if !breaker.allow(9, now) {
		t.Fatal("non-transient failures must not open fast transient circuit")
	}
}

func TestScoreAccountPoolCandidatesPrefersHealthyLowLoadAccount(t *testing.T) {
	stats := newAccountPoolRuntimeStats()
	badTTFT := 900
	goodTTFT := 120
	for i := 0; i < 5; i++ {
		stats.report(1, false, &badTTFT)
		stats.report(2, true, &goodTTFT)
	}

	inputs := []accountPoolCandidateInput{
		{
			Account: &Account{ID: 1, Priority: 0},
			Load:    &AccountLoadInfo{AccountID: 1, LoadRate: 80, WaitingCount: 5},
		},
		{
			Account: &Account{ID: 2, Priority: 0},
			Load:    &AccountLoadInfo{AccountID: 2, LoadRate: 10, WaitingCount: 0},
		},
	}

	scores := scoreAccountPoolCandidates(inputs, stats, defaultAccountPoolScoreWeights(), time.Now())
	if len(scores) != 2 {
		t.Fatalf("expected two candidates, got %d", len(scores))
	}
	var bad, good accountPoolCandidateScore
	for _, score := range scores {
		switch score.Account.ID {
		case 1:
			bad = score
		case 2:
			good = score
		}
	}
	if good.Score <= bad.Score {
		t.Fatalf("healthy low-load account should score higher: good=%v bad=%v", good.Score, bad.Score)
	}
}

func TestScoreAccountPoolCandidatesMissingLoadIsNeutral(t *testing.T) {
	inputs := []accountPoolCandidateInput{
		{Account: &Account{ID: 1, Priority: 0}},
		{Account: &Account{ID: 2, Priority: 0}, Load: &AccountLoadInfo{AccountID: 2, LoadRate: 100, WaitingCount: 10}},
	}
	scores := scoreAccountPoolCandidates(inputs, nil, defaultAccountPoolScoreWeights(), time.Now())
	if len(scores) != 2 {
		t.Fatalf("expected two candidates, got %d", len(scores))
	}
	if scores[0].LoadFactor != 0.5 {
		t.Fatalf("missing load must be neutral 0.5, got %v", scores[0].LoadFactor)
	}
}

func TestRankAccountPoolCandidatesStableForSeed(t *testing.T) {
	candidates := []accountPoolCandidateScore{
		{Account: &Account{ID: 1, Priority: 0}, Score: 3},
		{Account: &Account{ID: 2, Priority: 0}, Score: 3},
		{Account: &Account{ID: 3, Priority: 0}, Score: 2},
	}
	first := rankAccountPoolCandidates(candidates, 2, "request-abc")
	second := rankAccountPoolCandidates(candidates, 2, "request-abc")
	if len(first) != len(second) {
		t.Fatalf("rank lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Account.ID != second[i].Account.ID {
			t.Fatalf("ranking must be deterministic for same seed: %v vs %v", first, second)
		}
	}
	if first[2].Account.ID != 3 {
		t.Fatalf("candidate outside Top-K must remain after Top-K selections, got account %d", first[2].Account.ID)
	}
}
