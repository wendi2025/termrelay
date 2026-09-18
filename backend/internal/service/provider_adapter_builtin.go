package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type builtinProviderAdapter struct {
	name               string
	platform           string
	buildModelsRequest func(context.Context, *Account) (*http.Request, error)
	normalizeError     func(*http.Response, []byte) NormalizedProviderError
}

func (a *builtinProviderAdapter) Name() string {
	if a == nil {
		return ""
	}
	return a.name
}

func (a *builtinProviderAdapter) Supports(account *Account) bool {
	if a == nil || account == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(account.Platform), a.platform)
}

func (a *builtinProviderAdapter) ResolveModel(account *Account, requestedModel string) (string, error) {
	if account == nil {
		return "", errors.New("account is required")
	}
	if !a.Supports(account) {
		return "", errors.New("account is not supported by provider adapter")
	}
	return account.GetMappedModel(requestedModel), nil
}

func (a *builtinProviderAdapter) NormalizeError(resp *http.Response, body []byte) NormalizedProviderError {
	if a != nil && a.normalizeError != nil {
		return a.normalizeError(resp, body)
	}
	return normalizeGenericProviderError(resp, body)
}

func (a *builtinProviderAdapter) BuildModelsRequest(ctx context.Context, account *Account) (*http.Request, error) {
	if a == nil || a.buildModelsRequest == nil {
		return nil, errors.New("provider does not support model request building")
	}
	return a.buildModelsRequest(ctx, account)
}

func newAccountTestProviderAdapterRegistry(s *AccountTestService) *ProviderAdapterRegistry {
	if s == nil {
		return MustNewProviderAdapterRegistry()
	}

	return MustNewProviderAdapterRegistry(
		&builtinProviderAdapter{
			name:               PlatformAnthropic,
			platform:           PlatformAnthropic,
			buildModelsRequest: s.buildAnthropicUpstreamModelsRequest,
			normalizeError:     normalizeAnthropicProviderError,
		},
		&builtinProviderAdapter{
			name:               PlatformOpenAI,
			platform:           PlatformOpenAI,
			buildModelsRequest: s.buildOpenAIUpstreamModelsRequest,
			normalizeError:     normalizeOpenAIProviderError,
		},
		&builtinProviderAdapter{
			name:               PlatformGemini,
			platform:           PlatformGemini,
			buildModelsRequest: s.buildGeminiUpstreamModelsRequest,
			normalizeError:     normalizeGeminiProviderError,
		},
		&builtinProviderAdapter{
			name:               PlatformAntigravity,
			platform:           PlatformAntigravity,
			buildModelsRequest: s.buildAntigravityAPIKeyModelsRequest,
			normalizeError:     normalizeAntigravityProviderError,
		},
		&builtinProviderAdapter{
			name:               PlatformGrok,
			platform:           PlatformGrok,
			buildModelsRequest: s.buildGrokUpstreamModelsRequest,
			normalizeError:     normalizeGrokProviderError,
		},
	)
}
