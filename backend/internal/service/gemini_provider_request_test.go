package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/stretchr/testify/require"
)

func newGeminiAdapterTestService() *GeminiMessagesCompatService {
	return &GeminiMessagesCompatService{
		tokenProvider: &GeminiTokenProvider{},
		cfg:           &config.Config{},
	}
}

func TestGeminiGatewayProviderAdapterAPIKeyBuildsAIStudioRequestAndAuth(t *testing.T) {
	svc := newGeminiAdapterTestService()
	adapter := newGeminiGatewayProviderAdapter(svc)
	builder, ok := adapter.(ProviderRequestBuilder)
	require.True(t, ok)
	auth, ok := adapter.(ProviderAuthApplier)
	require.True(t, ok)
	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "secret-key",
		},
	}
	input := ProviderRequestInput{
		Account:  account,
		Protocol: ProviderProtocolChatCompletions,
		Endpoint: "streamGenerateContent",
		Model:    "gemini-2.5-flash",
		Body:     []byte(`{"tools":[{"googleSearch":{}}]}`),
		Stream:   true,
	}

	req, err := builder.BuildRequest(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, req.Method)
	require.Contains(t, req.URL.String(), "/v1beta/models/gemini-2.5-flash:streamGenerateContent?alt=sse")

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	tools, ok := payload["tools"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, tools)
	tool, ok := tools[0].(map[string]any)
	require.True(t, ok)
	_, hasSnake := tool["google_search"]
	_, hasCamel := tool["googleSearch"]
	require.True(t, hasSnake)
	require.False(t, hasCamel)

	require.NoError(t, auth.ApplyAuth(context.Background(), req, input))
	require.Equal(t, "secret-key", req.Header.Get("x-goog-api-key"))
	require.Empty(t, req.Header.Get("Authorization"))
}

func TestGeminiGatewayProviderAdapterOAuthCodeAssistWrapsAndAuthenticates(t *testing.T) {
	svc := newGeminiAdapterTestService()
	adapter := newGeminiGatewayProviderAdapter(svc)
	builder, ok := adapter.(ProviderRequestBuilder)
	require.True(t, ok)
	auth, ok := adapter.(ProviderAuthApplier)
	require.True(t, ok)
	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "ya29.adapter-test",
			"project_id":   "project-1",
		},
	}
	input := ProviderRequestInput{
		Account:  account,
		Protocol: ProviderProtocolAnthropic,
		Endpoint: "streamGenerateContent",
		Model:    "gemini-2.5-pro",
		Body:     []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`),
		Stream:   true,
	}

	req, err := builder.BuildRequest(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "https://cloudcode-pa.googleapis.com/v1internal:streamGenerateContent?alt=sse", req.URL.String())

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	var wrapped map[string]any
	require.NoError(t, json.Unmarshal(body, &wrapped))
	require.Equal(t, "gemini-2.5-pro", wrapped["model"])
	require.Equal(t, "project-1", wrapped["project"])
	require.NotNil(t, wrapped["request"])

	require.NoError(t, auth.ApplyAuth(context.Background(), req, input))
	require.Equal(t, "Bearer ya29.adapter-test", req.Header.Get("Authorization"))
	require.Equal(t, geminicli.GeminiCLIUserAgent, req.Header.Get("User-Agent"))
}

func TestGeminiGatewayProviderAdapterCountTokensForcesAIStudioForOAuth(t *testing.T) {
	svc := newGeminiAdapterTestService()
	adapter := newGeminiGatewayProviderAdapter(svc)
	builder, ok := adapter.(ProviderRequestBuilder)
	require.True(t, ok)
	auth, ok := adapter.(ProviderAuthApplier)
	require.True(t, ok)
	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "ya29.adapter-test",
			"project_id":   "project-1",
		},
	}
	input := ProviderRequestInput{
		Account:  account,
		Protocol: ProviderProtocolGemini,
		Endpoint: "countTokens",
		Model:    "gemini-2.5-flash",
		Body:     []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`),
	}

	req, err := builder.BuildRequest(context.Background(), input)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(req.URL.String(), "https://generativelanguage.googleapis.com/v1beta/models/"))
	require.Contains(t, req.URL.String(), ":countTokens")

	require.NoError(t, auth.ApplyAuth(context.Background(), req, input))
	require.Equal(t, "Bearer ya29.adapter-test", req.Header.Get("Authorization"))
	require.Empty(t, req.Header.Get("User-Agent"))
}

func TestGeminiGatewayProviderAdapterPreservesOAuthModelMappingSemantics(t *testing.T) {
	svc := newGeminiAdapterTestService()
	adapter := newGeminiGatewayProviderAdapter(svc)

	apiKey := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{"requested": "mapped"},
		},
	}
	mapped, err := adapter.ResolveModel(apiKey, "requested")
	require.NoError(t, err)
	require.Equal(t, "mapped", mapped)

	oauth := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"model_mapping": map[string]any{"requested": "mapped"},
		},
	}
	mapped, err = adapter.ResolveModel(oauth, "requested")
	require.NoError(t, err)
	require.Equal(t, "requested", mapped)
}

func TestGeminiGatewayProviderAdapterExposesRequestAndAuthCapabilities(t *testing.T) {
	svc := NewGeminiMessagesCompatService(nil, nil, nil, nil, &GeminiTokenProvider{}, nil, nil, nil, &config.Config{})
	caps, err := svc.providerAdapters.Capabilities(PlatformGemini)
	require.NoError(t, err)
	require.Contains(t, caps, ProviderCapabilityRequestPreparer)
	require.Contains(t, caps, ProviderCapabilityRequestBuilder)
	require.Contains(t, caps, ProviderCapabilityAuthApplier)
}

func TestGeminiGatewayProviderAdapterPrepareRequestResolvesOAuthToken(t *testing.T) {
	svc := newGeminiAdapterTestService()
	adapter, ok := newGeminiGatewayProviderAdapter(svc).(*geminiGatewayProviderAdapter)
	require.True(t, ok)
	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "ya29.prepared-token",
		},
	}
	input := ProviderRequestInput{
		Account:  account,
		Protocol: ProviderProtocolGemini,
		Endpoint: "generateContent",
		Model:    "gemini-2.5-flash",
		Body:     []byte(`{"contents":[]}`),
	}

	prepared, err := adapter.PrepareRequest(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "ya29.prepared-token", prepared.AuthToken)
}
