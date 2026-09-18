package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type capabilityTestAdapter struct {
	testProviderAdapter
}

func (a *capabilityTestAdapter) BuildModelsRequest(context.Context, *Account) (*http.Request, error) {
	return &http.Request{}, nil
}

func TestProviderAdapterDescriptorsAreDeterministicAndSafe(t *testing.T) {
	registry := MustNewProviderAdapterRegistry(
		&capabilityTestAdapter{testProviderAdapter: testProviderAdapter{name: "zeta", platform: "zeta"}},
		&testProviderAdapter{name: "alpha", platform: "alpha"},
	)

	descriptors := registry.Descriptors()
	require.Len(t, descriptors, 2)
	require.Equal(t, "alpha", descriptors[0].Name)
	require.Empty(t, descriptors[0].Capabilities)
	require.Equal(t, "zeta", descriptors[1].Name)
	require.Equal(t, []ProviderCapability{ProviderCapabilityModelsRequestBuilder}, descriptors[1].Capabilities)
}

func TestResolveProviderCapability(t *testing.T) {
	adapter := &capabilityTestAdapter{testProviderAdapter: testProviderAdapter{name: PlatformOpenAI, platform: PlatformOpenAI}}
	registry := MustNewProviderAdapterRegistry(adapter)

	builder, err := ResolveProviderCapability[ProviderModelsRequestBuilder](registry, &Account{Platform: PlatformOpenAI})
	require.NoError(t, err)
	require.Same(t, adapter, builder)

	_, err = ResolveProviderCapability[ProviderStreamingParser](registry, &Account{Platform: PlatformOpenAI})
	require.Error(t, err)
}

func TestBuiltinProviderCapabilitiesExposeModelRequestBoundary(t *testing.T) {
	registry := newAccountTestProviderAdapterRegistry(&AccountTestService{})
	for _, platform := range []string{
		PlatformAnthropic,
		PlatformOpenAI,
		PlatformGemini,
		PlatformAntigravity,
		PlatformGrok,
	} {
		capabilities, err := registry.Capabilities(platform)
		require.NoError(t, err)
		require.Contains(t, capabilities, ProviderCapabilityModelsRequestBuilder)
	}
}

func TestAnthropicProviderErrorNormalization(t *testing.T) {
	registry := newAccountTestProviderAdapterRegistry(&AccountTestService{})
	adapter, err := registry.Resolve(&Account{Platform: PlatformAnthropic})
	require.NoError(t, err)

	got := adapter.NormalizeError(
		&http.Response{StatusCode: 529, Header: http.Header{"Request-Id": []string{"anthropic_req"}}},
		[]byte(`{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`),
	)

	require.Equal(t, "overloaded_error", got.Code)
	require.Equal(t, "upstream_error", got.Type)
	require.Equal(t, "anthropic_req", got.RequestID)
	require.True(t, got.Retryable)
	require.True(t, got.Failover)
	require.True(t, got.Temporary)
}

func TestOpenAIInsufficientQuotaFailsOverWithoutSameAccountRetry(t *testing.T) {
	registry := newAccountTestProviderAdapterRegistry(&AccountTestService{})
	adapter, err := registry.Resolve(&Account{Platform: PlatformOpenAI})
	require.NoError(t, err)

	got := adapter.NormalizeError(
		&http.Response{StatusCode: http.StatusTooManyRequests, Header: make(http.Header)},
		[]byte(`{"error":{"message":"quota exceeded","type":"insufficient_quota","code":"insufficient_quota"}}`),
	)

	require.Equal(t, "insufficient_quota", got.Code)
	require.Equal(t, "rate_limit_error", got.Type)
	require.True(t, got.RateLimited)
	require.False(t, got.Retryable)
	require.True(t, got.Failover)
	require.False(t, got.Temporary)
}

func TestOpenAIContextLengthErrorIsNotTransient(t *testing.T) {
	registry := newAccountTestProviderAdapterRegistry(&AccountTestService{})
	adapter, err := registry.Resolve(&Account{Platform: PlatformOpenAI})
	require.NoError(t, err)

	got := adapter.NormalizeError(
		&http.Response{StatusCode: http.StatusBadRequest, Header: make(http.Header)},
		[]byte(`{"error":{"message":"too many tokens","type":"invalid_request_error","code":"context_length_exceeded"}}`),
	)

	require.Equal(t, "context_length_exceeded", got.Code)
	require.False(t, got.Retryable)
	require.False(t, got.Temporary)
	require.False(t, got.Failover)
}

func TestGeminiResourceExhaustedNormalization(t *testing.T) {
	registry := newAccountTestProviderAdapterRegistry(&AccountTestService{})
	adapter, err := registry.Resolve(&Account{Platform: PlatformGemini})
	require.NoError(t, err)

	got := adapter.NormalizeError(
		&http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"X-Goog-Request-Id": []string{"google_req"}}},
		[]byte(`{"error":{"code":429,"message":"quota","status":"RESOURCE_EXHAUSTED"}}`),
	)

	require.Equal(t, "RESOURCE_EXHAUSTED", got.Code)
	require.Equal(t, "google_req", got.RequestID)
	require.True(t, got.RateLimited)
	require.True(t, got.Retryable)
	require.True(t, got.Failover)
	require.True(t, got.Temporary)
}

func TestGrokUsesOpenAICompatibleErrorEnvelopeWithoutSharingRegistryIdentity(t *testing.T) {
	registry := newAccountTestProviderAdapterRegistry(&AccountTestService{})
	adapter, err := registry.Resolve(&Account{Platform: PlatformGrok})
	require.NoError(t, err)
	require.Equal(t, PlatformGrok, adapter.Name())

	got := adapter.NormalizeError(
		&http.Response{StatusCode: http.StatusUnauthorized, Header: make(http.Header)},
		[]byte(`{"error":{"message":"bad token","type":"authentication_error","code":"invalid_api_key"}}`),
	)

	require.Equal(t, "invalid_api_key", got.Code)
	require.True(t, got.AuthFailure)
	require.False(t, got.Retryable)
}
