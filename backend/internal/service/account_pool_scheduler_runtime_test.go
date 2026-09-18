package service

import (
	"testing"
	"time"
)

func TestAccountPoolSchedulerRankAndFeedback(t *testing.T) {
	scheduler := NewAccountPoolScheduler(1, AccountPoolScoreWeights{})
	now := time.Unix(1000, 0)
	fast := 100
	slow := 900

	for i := 0; i < 4; i++ {
		scheduler.Report(1, false, true, &slow, now)
		scheduler.Report(2, true, false, &fast, now)
	}

	ranked := scheduler.Rank([]accountPoolCandidateInput{
		{Account: &Account{ID: 1, Priority: 0}, Load: &AccountLoadInfo{AccountID: 1, LoadRate: 70, WaitingCount: 2}},
		{Account: &Account{ID: 2, Priority: 0}, Load: &AccountLoadInfo{AccountID: 2, LoadRate: 10}},
		{Account: &Account{ID: 3, Priority: 1}, Load: &AccountLoadInfo{AccountID: 3, LoadRate: 10}},
	}, "request-1", now)
	if len(ranked) != 3 {
		t.Fatalf("expected 3 ranked candidates, got %d", len(ranked))
	}
	if ranked[0].Account.ID != 2 {
		t.Fatalf("expected healthy low-load account first, got %d", ranked[0].Account.ID)
	}

	feedback := scheduler.Feedback(2)
	if feedback.Samples != 4 || feedback.ErrorRate != 0 || !feedback.HasTTFT {
		t.Fatalf("unexpected feedback: %+v", feedback)
	}
}

func TestAccountPoolSchedulerHalfOpenClaimOccursOnlyOnAllow(t *testing.T) {
	scheduler := NewAccountPoolScheduler(2, AccountPoolScoreWeights{})
	now := time.Unix(2000, 0)

	// Two transient failures open the fast circuit.
	scheduler.Report(7, false, true, nil, now)
	scheduler.Report(7, false, true, nil, now)

	// Ranking must be side-effect free and must not consume the half-open probe.
	_ = scheduler.Rank([]accountPoolCandidateInput{
		{Account: &Account{ID: 7, Priority: 0}},
		{Account: &Account{ID: 8, Priority: 0}},
	}, "request-2", now.Add(2*time.Second))

	if !scheduler.Allow(7, now.Add(2*time.Second)) {
		t.Fatal("expected first Allow after cooldown to claim half-open probe")
	}
	if scheduler.Allow(7, now.Add(2*time.Second)) {
		t.Fatal("expected second concurrent Allow to be blocked while probe is in flight")
	}

	scheduler.Report(7, true, false, nil, now.Add(2*time.Second))
	if !scheduler.Allow(7, now.Add(2*time.Second)) {
		t.Fatal("successful probe should close the circuit")
	}
}

func TestAccountPoolSchedulerDoesNotTripOnPermanentFailure(t *testing.T) {
	scheduler := NewAccountPoolScheduler(1, AccountPoolScoreWeights{})
	now := time.Unix(3000, 0)
	for i := 0; i < 5; i++ {
		scheduler.Report(9, false, false, nil, now)
	}
	if !scheduler.Allow(9, now) {
		t.Fatal("permanent/non-transient failures must not trip fast transient breaker")
	}
}
