package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

func TestAccountPoolSchedulerFeatureGateDefaultsOff(t *testing.T) {
	t.Setenv(gatewayAccountPoolEnabledEnv, "")
	if (&GatewayService{}).AccountPoolSchedulerEnabled() {
		t.Fatal("account-pool scheduler must default off")
	}
	t.Setenv(gatewayAccountPoolEnabledEnv, "true")
	if !(&GatewayService{}).AccountPoolSchedulerEnabled() {
		t.Fatal("explicit true should enable account-pool scheduler")
	}
}

func TestGatewayAccountPoolTopKBounds(t *testing.T) {
	t.Setenv(gatewayAccountPoolTopKEnv, "")
	if got := gatewayAccountPoolTopK(); got != defaultGatewayAccountPoolTopK {
		t.Fatalf("default top-k=%d want=%d", got, defaultGatewayAccountPoolTopK)
	}
	t.Setenv(gatewayAccountPoolTopKEnv, "999")
	if got := gatewayAccountPoolTopK(); got != 32 {
		t.Fatalf("top-k should clamp at 32, got %d", got)
	}
	t.Setenv(gatewayAccountPoolTopKEnv, "invalid")
	if got := gatewayAccountPoolTopK(); got != defaultGatewayAccountPoolTopK {
		t.Fatalf("invalid top-k should use default, got %d", got)
	}
}

func TestAccountPoolSelectionSeedPrefersRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.RequestID, "req-123")
	ctx = context.WithValue(ctx, ctxkey.ClientRequestID, "client-456")
	if got := accountPoolSelectionSeed(ctx, "session-789"); got != "req-123" {
		t.Fatalf("expected server request id seed, got %q", got)
	}
}

func TestAccountPoolSelectionSeedFallsBackToSession(t *testing.T) {
	if got := accountPoolSelectionSeed(context.Background(), "session-789"); got != "session-789" {
		t.Fatalf("expected session seed, got %q", got)
	}
}

func TestClassifyAccountPoolAttempt(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		report    bool
		success   bool
		transient bool
	}{
		{name: "success", report: true, success: true},
		{name: "client canceled", err: context.Canceled},
		{
			name: "429 uses persistent cooldown",
			err: &UpstreamFailoverError{
				StatusCode:        http.StatusTooManyRequests,
				NextAccountAction: NextAccountRetry,
			},
			report: true,
		},
		{
			name: "503 trips transient breaker",
			err: &UpstreamFailoverError{
				StatusCode:        http.StatusServiceUnavailable,
				NextAccountAction: NextAccountRetry,
			},
			report:    true,
			transient: true,
		},
		{
			name: "request scoped credential failure ignored",
			err: &UpstreamFailoverError{
				StatusCode:        http.StatusUnauthorized,
				Stage:             GatewayFailureStageAccountAuth,
				Scope:             GatewayFailureScopeRequest,
				NextAccountAction: NextAccountStop,
			},
		},
		{
			name: "account scoped credential failure counted",
			err: &UpstreamFailoverError{
				StatusCode:        http.StatusUnauthorized,
				Stage:             GatewayFailureStageAccountAuth,
				Scope:             GatewayFailureScopeAccount,
				NextAccountAction: NextAccountRetry,
			},
			report: true,
		},
		{
			name:      "sanitized transport error stays transient",
			err:       errors.New("upstream request failed: connection reset"),
			report:    true,
			transient: true,
		},
		{name: "unknown internal error ignored", err: errors.New("internal parse error")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report, success, transient := classifyAccountPoolAttempt(tc.err)
			if report != tc.report || success != tc.success || transient != tc.transient {
				t.Fatalf("got (%v,%v,%v), want (%v,%v,%v)", report, success, transient, tc.report, tc.success, tc.transient)
			}
		})
	}
}
