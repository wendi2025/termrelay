package service

import (
	"testing"
	"time"
)

func TestStaticWeightDoesNotOverrideHealthyTopKGate(t *testing.T) {
	now := time.Unix(5000, 0)
	scheduler := NewAccountPoolScheduler(1, AccountPoolScoreWeights{})
	fast := 100
	slow := 1500
	for i := 0; i < 5; i++ {
		scheduler.Report(1, true, false, &fast, now)
		scheduler.Report(2, false, true, &slow, now)
	}

	ranked := scheduler.Rank([]accountPoolCandidateInput{
		{
			Account: &Account{ID: 1, Priority: 0, Extra: map[string]any{accountPoolWeightExtraKey: 1.0}},
			Load:    &AccountLoadInfo{AccountID: 1, LoadRate: 10},
		},
		{
			Account: &Account{ID: 2, Priority: 0, Extra: map[string]any{accountPoolWeightExtraKey: 1000.0}},
			Load:    &AccountLoadInfo{AccountID: 2, LoadRate: 90, WaitingCount: 5},
		},
	}, "health-before-weight", now)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(ranked))
	}
	if ranked[0].Account.ID != 1 {
		t.Fatalf("unhealthy account must not buy its way into Top-K with weight; got %d", ranked[0].Account.ID)
	}
}

func TestStaticWeightNeverOverridesPriorityTier(t *testing.T) {
	now := time.Unix(6000, 0)
	scheduler := NewAccountPoolScheduler(3, AccountPoolScoreWeights{})

	ranked := scheduler.Rank([]accountPoolCandidateInput{
		{
			Account: &Account{ID: 10, Priority: 0, Extra: map[string]any{accountPoolWeightExtraKey: 1.0}},
			Load:    &AccountLoadInfo{AccountID: 10, LoadRate: 40},
		},
		{
			Account: &Account{ID: 20, Priority: 1, Extra: map[string]any{accountPoolWeightExtraKey: 1000.0}},
			Load:    &AccountLoadInfo{AccountID: 20, LoadRate: 0},
		},
	}, "priority-before-weight", now)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(ranked))
	}
	if ranked[0].Account.ID != 10 {
		t.Fatalf("lower-priority tier must never outrank priority 0 through weight; got %d", ranked[0].Account.ID)
	}
}
