package service

import (
	"testing"
	"time"
)

func TestAccountPoolSchedulerAbandonReleasesHalfOpenProbe(t *testing.T) {
	now := time.Unix(4000, 0)
	scheduler := NewAccountPoolScheduler(1, AccountPoolScoreWeights{})
	scheduler.Report(11, false, true, nil, now)
	scheduler.Report(11, false, true, nil, now)

	probeAt := now.Add(2 * time.Second)
	if !scheduler.Allow(11, probeAt) {
		t.Fatal("expected first half-open probe to be allowed")
	}
	if scheduler.Allow(11, probeAt) {
		t.Fatal("probe should be reserved until reported or abandoned")
	}

	scheduler.Abandon(11)
	if !scheduler.Allow(11, probeAt) {
		t.Fatal("abandon should release the unused half-open probe")
	}
}
