package service

import (
	"sync/atomic"
	"time"
)

// CodexMetrics holds atomic counters for Codex proxy observability.
type CodexMetrics struct {
	RequestsTotal           atomic.Int64
	RequestsSuccess         atomic.Int64
	RequestsClientError     atomic.Int64
	RequestsServerError     atomic.Int64
	RequestsNotModified     atomic.Int64
	CacheHits               atomic.Int64
	CacheStales             atomic.Int64
	CacheMisses             atomic.Int64
	AccountSwitchesTotal    atomic.Int64
	AccountSwitchesOK       atomic.Int64
	AccountSwitchesFail     atomic.Int64
	LatencyMsTotal          atomic.Int64
	UpstreamErrors401       atomic.Int64
	UpstreamErrors429       atomic.Int64
	UpstreamErrors5xx       atomic.Int64
	UpstreamErrorsNet       atomic.Int64
	AgentIdentityRecoveries atomic.Int64
	AgentIdentityFailed     atomic.Int64
}

var codexMetrics CodexMetrics

func CodexMetricsInstance() *CodexMetrics {
	return &codexMetrics
}

func (m *CodexMetrics) RecordRequest(statusCode int) {
	m.RequestsTotal.Add(1)
	switch {
	case statusCode >= 200 && statusCode < 300:
		m.RequestsSuccess.Add(1)
	case statusCode == 304:
		m.RequestsNotModified.Add(1)
	case statusCode >= 400 && statusCode < 500:
		m.RequestsClientError.Add(1)
	case statusCode >= 500:
		m.RequestsServerError.Add(1)
	}
}

func (m *CodexMetrics) RecordCacheHit(fresh bool) {
	if fresh {
		m.CacheHits.Add(1)
	} else {
		m.CacheStales.Add(1)
	}
}

func (m *CodexMetrics) RecordCacheMiss() {
	m.CacheMisses.Add(1)
}

func (m *CodexMetrics) RecordAccountSwitch(success bool) {
	m.AccountSwitchesTotal.Add(1)
	if success {
		m.AccountSwitchesOK.Add(1)
	} else {
		m.AccountSwitchesFail.Add(1)
	}
}

func (m *CodexMetrics) RecordLatency(ms int64) {
	m.LatencyMsTotal.Add(ms)
}

func (m *CodexMetrics) RecordUpstreamError(statusCode int) {
	switch statusCode {
	case 401:
		m.UpstreamErrors401.Add(1)
	case 429:
		m.UpstreamErrors429.Add(1)
	default:
		if statusCode >= 500 {
			m.UpstreamErrors5xx.Add(1)
		}
	}
}

func (m *CodexMetrics) RecordUpstreamNetworkError() {
	m.UpstreamErrorsNet.Add(1)
}

func (m *CodexMetrics) RecordAgentIdentityRecovery(success bool) {
	if success {
		m.AgentIdentityRecoveries.Add(1)
	} else {
		m.AgentIdentityFailed.Add(1)
	}
}

type CodexRequestLifecycle struct {
	start time.Time
}

func NewCodexRequestLifecycle() *CodexRequestLifecycle {
	return &CodexRequestLifecycle{start: time.Now()}
}

func (l *CodexRequestLifecycle) LatencyMs() int64 {
	return time.Since(l.start).Milliseconds()
}
