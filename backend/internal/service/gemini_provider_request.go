package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

const geminiProviderRequestIDHeader = "x-request-id"

type geminiGatewayProviderAdapter struct {
	service *GeminiMessagesCompatService
}

func newGeminiGatewayProviderAdapter(service *GeminiMessagesCompatService) ProviderAdapter {
	return &geminiGatewayProviderAdapter{service: service}
}

func (a *geminiGatewayProviderAdapter) Name() string { return PlatformGemini }

func (a *geminiGatewayProviderAdapter) Supports(account *Account) bool {
	return account != nil && strings.EqualFold(strings.TrimSpace(account.Platform), PlatformGemini)
}

func (a *geminiGatewayProviderAdapter) ResolveModel(account *Account, requestedModel string) (string, error) {
	if account == nil {
		return "", errors.New("account is required")
	}
	if !a.Supports(account) {
		return "", errors.New("account is not supported by gemini provider adapter")
	}
	if strings.TrimSpace(requestedModel) == "" {
		return "", errors.New("gemini model is required")
	}

	switch account.Type {
	case AccountTypeAPIKey, AccountTypeServiceAccount:
		return account.GetMappedModel(requestedModel), nil
	case AccountTypeOAuth:
		// Preserve existing Gemini OAuth behavior: Code Assist / AI Studio OAuth
		// historically forwards the requested model without account mapping.
		return requestedModel, nil
	default:
		return "", fmt.Errorf("unsupported account type: %s", account.Type)
	}
}

func (a *geminiGatewayProviderAdapter) PrepareRequest(ctx context.Context, input ProviderRequestInput) (ProviderRequestInput, error) {
	if a == nil || a.service == nil {
		return input, errors.New("gemini provider adapter is not configured")
	}
	account := input.Account
	if account == nil {
		return input, errors.New("gemini account is required")
	}
	if !a.Supports(account) {
		return input, errors.New("account is not supported by gemini provider adapter")
	}

	switch account.Type {
	case AccountTypeAPIKey:
		return input, nil
	case AccountTypeOAuth, AccountTypeServiceAccount:
		if a.service.tokenProvider == nil {
			return input, errors.New("gemini token provider not configured")
		}
		accessToken, err := a.service.tokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return input, err
		}
		input.AuthToken = accessToken
		return input, nil
	default:
		return input, fmt.Errorf("unsupported account type: %s", account.Type)
	}
}

func (a *geminiGatewayProviderAdapter) NormalizeError(resp *http.Response, body []byte) NormalizedProviderError {
	return normalizeGeminiProviderError(resp, body)
}

func (a *geminiGatewayProviderAdapter) BuildRequest(ctx context.Context, input ProviderRequestInput) (*http.Request, error) {
	if a == nil || a.service == nil {
		return nil, errors.New("gemini provider adapter is not configured")
	}
	account := input.Account
	if account == nil {
		return nil, errors.New("gemini account is required")
	}
	if !a.Supports(account) {
		return nil, errors.New("account is not supported by gemini provider adapter")
	}
	if strings.TrimSpace(input.Model) == "" {
		return nil, errors.New("gemini model is required")
	}

	switch input.Protocol {
	case ProviderProtocolAnthropic, ProviderProtocolChatCompletions, ProviderProtocolResponses, ProviderProtocolGemini:
		// Supported wire-protocol contexts. The request body has already been
		// converted to Gemini native shape by the compatibility layer.
	default:
		return nil, fmt.Errorf("unsupported gemini provider protocol: %s", input.Protocol)
	}

	action := strings.TrimSpace(input.Endpoint)
	if action == "" {
		action = "generateContent"
		if input.Stream {
			action = "streamGenerateContent"
		}
	}
	if _, ok := geminiAIStudioActions[action]; !ok {
		return nil, fmt.Errorf("unsupported gemini action: %s", action)
	}

	method := strings.TrimSpace(input.Method)
	if method == "" {
		method = http.MethodPost
	}
	if method != http.MethodPost {
		return nil, fmt.Errorf("unsupported gemini request method: %s", method)
	}

	bodyForREST := input.Body
	if input.Protocol != ProviderProtocolGemini {
		bodyForREST = normalizeGeminiRequestForAIStudio(input.Body)
	}

	forceAIStudio := input.Protocol == ProviderProtocolGemini && action == "countTokens"

	switch account.Type {
	case AccountTypeAPIKey:
		baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
		normalizedBaseURL, err := a.service.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		fullURL, err := buildGeminiAIStudioModelActionURL(normalizedBaseURL, input.Model, action, input.Stream)
		if err != nil {
			return nil, err
		}
		return http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(bodyForREST))

	case AccountTypeOAuth:
		projectID := strings.TrimSpace(account.GetCredential("project_id"))
		if projectID != "" && !forceAIStudio {
			baseURL, err := a.service.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
			if err != nil {
				return nil, err
			}
			fullURL := fmt.Sprintf("%s/v1internal:%s", strings.TrimRight(baseURL, "/"), action)
			if input.Stream {
				fullURL += "?alt=sse"
			}

			var inner any
			if err := json.Unmarshal(input.Body, &inner); err != nil {
				return nil, fmt.Errorf("failed to parse gemini request: %w", err)
			}
			wrappedBytes, err := json.Marshal(map[string]any{
				"model":   input.Model,
				"project": projectID,
				"request": inner,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to wrap gemini request: %w", err)
			}
			return http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(wrappedBytes))
		}

		baseURL := account.GetGeminiBaseURL(geminicli.AIStudioBaseURL)
		normalizedBaseURL, err := a.service.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		fullURL, err := buildGeminiAIStudioModelActionURL(normalizedBaseURL, input.Model, action, input.Stream)
		if err != nil {
			return nil, err
		}
		return http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(bodyForREST))

	case AccountTypeServiceAccount:
		fullURL, err := buildVertexGeminiURL(
			account.VertexProjectID(),
			account.VertexLocation(input.Model),
			input.Model,
			action,
			input.Stream,
		)
		if err != nil {
			return nil, err
		}
		return http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(bodyForREST))

	default:
		return nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}
}

func (a *geminiGatewayProviderAdapter) ApplyAuth(ctx context.Context, req *http.Request, input ProviderRequestInput) error {
	if a == nil || a.service == nil {
		return errors.New("gemini provider adapter is not configured")
	}
	if req == nil {
		return errors.New("gemini upstream request is required")
	}
	account := input.Account
	if account == nil {
		return errors.New("gemini account is required")
	}

	req.Header.Set("Content-Type", "application/json")
	switch account.Type {
	case AccountTypeAPIKey:
		apiKey := account.GetCredential("api_key")
		if strings.TrimSpace(apiKey) == "" {
			return errors.New("gemini api_key not configured")
		}
		req.Header.Set("x-goog-api-key", apiKey)
		return nil

	case AccountTypeOAuth, AccountTypeServiceAccount:
		accessToken := input.AuthToken
		if strings.TrimSpace(accessToken) == "" {
			if a.service.tokenProvider == nil {
				return errors.New("gemini token provider not configured")
			}
			var err error
			accessToken, err = a.service.tokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return err
			}
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		forceAIStudio := input.Protocol == ProviderProtocolGemini && strings.TrimSpace(input.Endpoint) == "countTokens"
		if account.Type == AccountTypeOAuth && strings.TrimSpace(account.GetCredential("project_id")) != "" && !forceAIStudio {
			req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
		}
		return nil

	default:
		return fmt.Errorf("unsupported account type: %s", account.Type)
	}
}

func (s *GeminiMessagesCompatService) geminiProviderAdapterRegistry() *ProviderAdapterRegistry {
	if s != nil && s.providerAdapters != nil {
		return s.providerAdapters
	}
	return MustNewProviderAdapterRegistry(newGeminiGatewayProviderAdapter(s))
}

func (s *GeminiMessagesCompatService) resolveGeminiProviderModel(account *Account, requestedModel string) (string, error) {
	adapter, err := s.geminiProviderAdapterRegistry().Resolve(account)
	if err != nil {
		return "", err
	}
	return adapter.ResolveModel(account, requestedModel)
}

func (s *GeminiMessagesCompatService) newGeminiProviderRequestFactory(input ProviderRequestInput) (func(context.Context) (*http.Request, string, error), string, error) {
	registry := s.geminiProviderAdapterRegistry()
	adapter, err := registry.Resolve(input.Account)
	if err != nil {
		return nil, "", err
	}
	builder, ok := adapter.(ProviderRequestBuilder)
	if !ok {
		return nil, "", fmt.Errorf("provider %s does not implement request builder", adapter.Name())
	}
	auth, ok := adapter.(ProviderAuthApplier)
	if !ok {
		return nil, "", fmt.Errorf("provider %s does not implement auth applier", adapter.Name())
	}

	return func(ctx context.Context) (*http.Request, string, error) {
		preparedInput := input
		if preparer, ok := adapter.(ProviderRequestPreparer); ok {
			var prepareErr error
			preparedInput, prepareErr = preparer.PrepareRequest(ctx, input)
			if prepareErr != nil {
				return nil, "", prepareErr
			}
		}

		req, err := builder.BuildRequest(ctx, preparedInput)
		if err != nil {
			return nil, "", err
		}
		if err := auth.ApplyAuth(ctx, req, preparedInput); err != nil {
			return nil, "", err
		}
		return req, geminiProviderRequestIDHeader, nil
	}, geminiProviderRequestIDHeader, nil
}
