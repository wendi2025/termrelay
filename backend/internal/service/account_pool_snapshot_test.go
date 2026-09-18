package service

import "testing"

func TestAccountPoolHealthStatus(t *testing.T) {
	tests := []struct {
		name   string
		health AccountPoolHealthSnapshot
		want   string
	}{
		{name: "unknown", health: AccountPoolHealthSnapshot{CircuitState: "closed"}, want: "unknown"},
		{name: "healthy", health: AccountPoolHealthSnapshot{CircuitState: "closed", Samples: 10, ErrorRate: 0.05}, want: "healthy"},
		{name: "degraded error rate", health: AccountPoolHealthSnapshot{CircuitState: "closed", Samples: 10, ErrorRate: 0.25}, want: "degraded"},
		{name: "degraded failure", health: AccountPoolHealthSnapshot{CircuitState: "closed", Samples: 10, ConsecutiveFailures: 1}, want: "degraded"},
		{name: "unhealthy", health: AccountPoolHealthSnapshot{CircuitState: "closed", Samples: 10, ErrorRate: 0.60}, want: "unhealthy"},
		{name: "cooling", health: AccountPoolHealthSnapshot{CircuitState: "open", Samples: 10}, want: "cooling"},
		{name: "half open", health: AccountPoolHealthSnapshot{CircuitState: "half_open", Samples: 10}, want: "probing"},
		{name: "probe ready", health: AccountPoolHealthSnapshot{CircuitState: "probe_ready", Samples: 10}, want: "probing"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := accountPoolHealthStatus(tc.health); got != tc.want {
				t.Fatalf("accountPoolHealthStatus()=%q want=%q", got, tc.want)
			}
		})
	}
}
