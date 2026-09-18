package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestAccountPoolProviderFeedbackCancelReleasesHalfOpenProbe(t *testing.T) {
	t.Setenv(gatewayAccountPoolEnabledEnv, "true")
	svc := &GatewayService{}
	accountID := int64(88001)
	scheduler := svc.accountPoolScheduler()
	now := time.Now()
	scheduler.Report(accountID, false, true, nil, now.Add(-3*time.Second))
	scheduler.Report(accountID, false, true, nil, now.Add(-2*time.Second))
	probeAt := now.Add(2 * time.Second)
	if !scheduler.Allow(accountID, probeAt) {
		t.Fatal("expected half-open probe")
	}
	if scheduler.Allow(accountID, probeAt) {
		t.Fatal("expected single half-open probe")
	}
	svc.ReportAccountPoolOutcome(accountID, nil, context.Canceled)
	if !scheduler.Allow(accountID, probeAt) {
		t.Fatal("cancel must release half-open probe")
	}
}

func TestAccountPoolProviderFeedbackAdmissionHonorsOpenCircuit(t *testing.T) {
	t.Setenv(gatewayAccountPoolEnabledEnv, "true")
	svc := &GatewayService{}
	accountID := int64(88002)
	svc.ReportAccountPoolOutcome(accountID, nil, &UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable})
	svc.ReportAccountPoolOutcome(accountID, nil, &UpstreamFailoverError{StatusCode: http.StatusBadGateway})
	if svc.BeginAccountPoolAttempt(accountID) {
		t.Fatal("open circuit must reject provider-specific attempt")
	}
}

func TestAccountPoolProviderFeedback429DoesNotTripFastBreaker(t *testing.T) {
	t.Setenv(gatewayAccountPoolEnabledEnv, "true")
	svc := &GatewayService{}
	accountID := int64(88003)
	svc.ReportAccountPoolOutcome(accountID, nil, &UpstreamFailoverError{StatusCode: http.StatusTooManyRequests})
	svc.ReportAccountPoolOutcome(accountID, nil, &UpstreamFailoverError{StatusCode: http.StatusTooManyRequests})
	if !svc.BeginAccountPoolAttempt(accountID) {
		t.Fatal("429 must not trip fast breaker")
	}
}

func TestAccountPoolProviderFeedbackTransportFailureTripsFastBreaker(t *testing.T) {
	t.Setenv(gatewayAccountPoolEnabledEnv, "true")
	svc := &GatewayService{}
	accountID := int64(88004)
	svc.ReportAccountPoolOutcome(accountID, nil, errors.New("upstream request failed: connection reset"))
	svc.ReportAccountPoolOutcome(accountID, nil, errors.New("upstream request failed: connection reset"))
	if svc.BeginAccountPoolAttempt(accountID) {
		t.Fatal("transport failures should open fast breaker")
	}
}
