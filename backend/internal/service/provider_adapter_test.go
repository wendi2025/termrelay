package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type testProviderAdapter struct {
	name     string
	platform string
}

func (a *testProviderAdapter) Name() string { return a.name }

func (a *testProviderAdapter) Supports(account *Account) bool {
	return account != nil && account.Platform == a.platform
}

func (a *testProviderAdapter) ResolveModel(account *Account, requestedModel string) (string, error) {
	if account == nil {
		return "", nil
	}
	return account.GetMappedModel(requestedModel), nil
}

func (a *testProviderAdapter) NormalizeError(resp *http.Response, body []byte) NormalizedProviderError {
	return normalizeGenericProviderError(resp, body)
}

func TestProviderAdapterRegistryResolve(t *testing.T) {
	openAI := &testProviderAdapter{name: PlatformOpenAI, platform: PlatformOpenAI}
	anthropic := &testProviderAdapter{name: PlatformAnthropic, platform: PlatformAnthropic}
	registry := MustNewProviderAdapterRegistry(openAI, anthropic)

	got, err := registry.Resolve(&Account{Platform: PlatformOpenAI})
	require.NoError(t, err)
	require.Same(t, openAI, got)

	got, err = registry.Resolve(&Account{Platform: PlatformAnthropic})
	require.NoError(t, err)
	require.Same(t, anthropic, got)

	_, err = registry.Resolve(&Account{Platform: PlatformComposite})
	require.Error(t, err)
}

func TestProviderAdapterRegistryRejectsAmbiguousMatches(t *testing.T) {
	registry := MustNewProviderAdapterRegistry(
		&testProviderAdapter{name: "openai-primary", platform: PlatformOpenAI},
		&testProviderAdapter{name: "openai-compatible", platform: PlatformOpenAI},
	)

	adapter, err := registry.Resolve(&Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey})
	require.Nil(t, adapter)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ambiguous provider adapter match")
	require.Contains(t, err.Error(), "openai-primary")
	require.Contains(t, err.Error(), "openai-compatible")
}

func TestProviderAdapterRegistryRejectsDuplicateNames(t *testing.T) {
	_, err := NewProviderAdapterRegistry(
		&testProviderAdapter{name: "OpenAI", platform: PlatformOpenAI},
		&testProviderAdapter{name: " openai ", platform: PlatformAnthropic},
	)
	require.Error(t, err)
}

func TestBuiltinProviderAdaptersCoverConcretePlatforms(t *testing.T) {
	registry := newAccountTestProviderAdapterRegistry(&AccountTestService{})
	for _, platform := range []string{
		PlatformAnthropic,
		PlatformOpenAI,
		PlatformGemini,
		PlatformAntigravity,
		PlatformGrok,
	} {
		adapter, err := registry.Resolve(&Account{Platform: platform})
		require.NoError(t, err, platform)
		require.Equal(t, platform, adapter.Name())
		_, ok := adapter.(ProviderModelsRequestBuilder)
		require.True(t, ok, platform)
	}
}

func TestBuiltinProviderAdapterResolveModelUsesAccountMapping(t *testing.T) {
	registry := newAccountTestProviderAdapterRegistry(&AccountTestService{})
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"public-model": "upstream-model",
			},
		},
	}

	adapter, err := registry.Resolve(account)
	require.NoError(t, err)
	mapped, err := adapter.ResolveModel(account, "public-model")
	require.NoError(t, err)
	require.Equal(t, "upstream-model", mapped)
}

func TestNormalizeGenericProviderError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header: http.Header{
			"X-Request-Id": []string{"req_123"},
		},
	}
	got := normalizeGenericProviderError(resp, []byte(`{"error":{"code":"rate_limit_exceeded","message":"rate limited"}}`))

	require.Equal(t, http.StatusTooManyRequests, got.StatusCode)
	require.Equal(t, "rate_limit_exceeded", got.Code)
	require.Equal(t, "req_123", got.RequestID)
	require.Equal(t, "rate_limit_error", got.Type)
	require.Equal(t, "rate limited", got.Message)
	require.True(t, got.RateLimited)
	require.True(t, got.Retryable)
	require.True(t, got.Failover)
	require.True(t, got.Temporary)
	require.False(t, got.AuthFailure)
}

func TestBuildUpstreamModelsRequestRejectsUnknownProviderThroughRegistry(t *testing.T) {
	svc := &AccountTestService{}
	_, err := svc.buildUpstreamModelsRequest(t.Context(), &Account{Platform: "unknown", Type: AccountTypeAPIKey})
	require.Error(t, err)

	var syncErr *UpstreamModelSyncError
	require.ErrorAs(t, err, &syncErr)
	require.Equal(t, UpstreamModelSyncErrorUnsupported, syncErr.Kind)
}
