package service

import (
	"testing"
	"time"
)

func TestAccountSchedulingWeightFromExtra(t *testing.T) {
	cases := []struct {
		name  string
		extra map[string]any
		want  float64
	}{
		{name: "default", extra: nil, want: 1},
		{name: "float", extra: map[string]any{accountPoolWeightExtraKey: 3.5}, want: 3.5},
		{name: "string", extra: map[string]any{accountPoolWeightExtraKey: "2"}, want: 2},
		{name: "invalid zero", extra: map[string]any{accountPoolWeightExtraKey: 0}, want: 1},
		{name: "invalid negative", extra: map[string]any{accountPoolWeightExtraKey: -4}, want: 1},
		{name: "clamped", extra: map[string]any{accountPoolWeightExtraKey: 5000}, want: 1000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{Extra: tc.extra}
			if got := account.SchedulingWeight(); got != tc.want {
				t.Fatalf("SchedulingWeight()=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestAccountPoolSchedulerWeightAffectsRanking(t *testing.T) {
	now := time.Unix(1000, 0)
	scheduler := NewAccountPoolScheduler(2, AccountPoolScoreWeights{})
	inputs := []accountPoolCandidateInput{
		{Account: &Account{ID: 1, Priority: 0, Extra: map[string]any{accountPoolWeightExtraKey: 1.0}}},
		{Account: &Account{ID: 2, Priority: 0, Extra: map[string]any{accountPoolWeightExtraKey: 100.0}}},
	}

	ranked := scheduler.Rank(inputs, "weighted-request", now)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(ranked))
	}
	if ranked[0].Account.ID != 2 {
		t.Fatalf("expected heavily weighted account first, got %d", ranked[0].Account.ID)
	}
}

func TestAccountPoolSchedulerHealthSnapshot(t *testing.T) {
	now := time.Unix(2000, 0)
	scheduler := NewAccountPoolScheduler(1, AccountPoolScoreWeights{})

	scheduler.Report(7, false, true, nil, now)
	scheduler.Report(7, false, true, nil, now.Add(time.Second))

	health := scheduler.Health(7, now.Add(time.Second))
	if health.ConsecutiveFailures != 2 {
		t.Fatalf("expected 2 consecutive failures, got %d", health.ConsecutiveFailures)
	}
	if health.LastFailureAt == nil || !health.LastFailureAt.Equal(now.Add(time.Second)) {
		t.Fatalf("unexpected last failure timestamp: %+v", health.LastFailureAt)
	}
	if health.CircuitState != "open" {
		t.Fatalf("expected open circuit, got %q", health.CircuitState)
	}
	if health.CooldownUntil == nil {
		t.Fatal("expected cooldown deadline")
	}

	scheduler.Report(7, true, false, nil, now.Add(3*time.Second))
	health = scheduler.Health(7, now.Add(3*time.Second))
	if health.ConsecutiveFailures != 0 {
		t.Fatalf("success should reset consecutive failures, got %d", health.ConsecutiveFailures)
	}
	if health.LastSuccessAt == nil || !health.LastSuccessAt.Equal(now.Add(3*time.Second)) {
		t.Fatalf("unexpected last success timestamp: %+v", health.LastSuccessAt)
	}
	if health.CircuitState != "closed" {
		t.Fatalf("success should close circuit, got %q", health.CircuitState)
	}
}

func TestNormalizeAccountPoolSchedulingWeight(t *testing.T) {
	t.Run("explicit typed value wins", func(t *testing.T) {
		weight := 7.5
		extra, err := NormalizeAccountPoolSchedulingWeight(map[string]any{
			accountPoolWeightExtraKey: "2",
			"keep":                    true,
		}, &weight)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := extra[accountPoolWeightExtraKey]; got != 7.5 {
			t.Fatalf("weight=%v want=7.5", got)
		}
		if extra["keep"] != true {
			t.Fatal("unrelated extra field was not preserved")
		}
	})

	t.Run("legacy extra string normalized", func(t *testing.T) {
		extra, err := NormalizeAccountPoolSchedulingWeight(map[string]any{accountPoolWeightExtraKey: "3.25"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := extra[accountPoolWeightExtraKey]; got != 3.25 {
			t.Fatalf("weight=%v want=3.25", got)
		}
	})

	for _, invalid := range []float64{0, -1, 1000.1} {
		invalid := invalid
		t.Run("reject invalid typed weight", func(t *testing.T) {
			if _, err := NormalizeAccountPoolSchedulingWeight(nil, &invalid); err == nil {
				t.Fatalf("expected error for weight %v", invalid)
			}
		})
	}
}
