package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type providerErrorEnvelope struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Error   struct {
		Code    any    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func decodeProviderErrorEnvelope(body []byte) providerErrorEnvelope {
	var envelope providerErrorEnvelope
	_ = json.Unmarshal(body, &envelope)
	return envelope
}

func providerErrorCodeString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

func normalizeAnthropicProviderError(resp *http.Response, body []byte) NormalizedProviderError {
	normalized := normalizeGenericProviderError(resp, body)
	envelope := decodeProviderErrorEnvelope(body)

	providerType := strings.TrimSpace(envelope.Error.Type)
	if providerType == "" && strings.EqualFold(strings.TrimSpace(envelope.Type), "error") {
		providerType = strings.TrimSpace(envelope.Error.Status)
	}
	if providerType != "" {
		normalized.Code = providerType
	}

	switch strings.ToLower(providerType) {
	case "rate_limit_error":
		normalized.Type = "rate_limit_error"
		normalized.RateLimited = true
		normalized.Retryable = true
		normalized.Failover = true
		normalized.Temporary = true
	case "overloaded_error", "api_error":
		normalized.Type = "upstream_error"
		normalized.Retryable = true
		normalized.Failover = true
		normalized.Temporary = true
	case "authentication_error", "permission_error":
		normalized.Type = "authentication_error"
		normalized.AuthFailure = true
		normalized.Retryable = false
		normalized.Failover = true
		normalized.Temporary = false
	}
	return normalized
}

func normalizeOpenAIProviderError(resp *http.Response, body []byte) NormalizedProviderError {
	normalized := normalizeGenericProviderError(resp, body)
	envelope := decodeProviderErrorEnvelope(body)

	if code := providerErrorCodeString(envelope.Error.Code); code != "" {
		normalized.Code = code
	}
	providerType := strings.TrimSpace(envelope.Error.Type)
	if providerType != "" && normalized.Type == "" {
		normalized.Type = providerType
	}

	code := strings.ToLower(normalized.Code)
	switch code {
	case "rate_limit_exceeded", "insufficient_quota":
		normalized.Type = "rate_limit_error"
		normalized.RateLimited = true
		// insufficient_quota is account state, not a transient same-account retry.
		if code == "insufficient_quota" {
			normalized.Retryable = false
		}
		normalized.Failover = true
		normalized.Temporary = code != "insufficient_quota"
	case "server_error", "temporarily_unavailable", "service_unavailable":
		normalized.Type = "upstream_error"
		normalized.Retryable = true
		normalized.Failover = true
		normalized.Temporary = true
	case "invalid_api_key", "invalid_authentication", "token_revoked", "token_invalidated":
		normalized.Type = "authentication_error"
		normalized.AuthFailure = true
		normalized.Retryable = false
		normalized.Failover = true
		normalized.Temporary = false
	case "context_length_exceeded", "model_not_found", "invalid_request_error":
		normalized.Retryable = false
		normalized.Temporary = false
	}
	return normalized
}

func normalizeGeminiProviderError(resp *http.Response, body []byte) NormalizedProviderError {
	normalized := normalizeGenericProviderError(resp, body)
	envelope := decodeProviderErrorEnvelope(body)

	status := strings.TrimSpace(envelope.Error.Status)
	if status == "" {
		status = providerErrorCodeString(envelope.Error.Code)
	}
	if status != "" {
		normalized.Code = status
	}

	switch strings.ToUpper(status) {
	case "RESOURCE_EXHAUSTED":
		normalized.Type = "rate_limit_error"
		normalized.RateLimited = true
		normalized.Retryable = true
		normalized.Failover = true
		normalized.Temporary = true
	case "UNAVAILABLE", "INTERNAL", "DEADLINE_EXCEEDED", "ABORTED":
		normalized.Type = "upstream_error"
		normalized.Retryable = true
		normalized.Failover = true
		normalized.Temporary = true
	case "UNAUTHENTICATED", "PERMISSION_DENIED":
		normalized.Type = "authentication_error"
		normalized.AuthFailure = true
		normalized.Retryable = false
		normalized.Failover = true
		normalized.Temporary = false
	case "INVALID_ARGUMENT", "NOT_FOUND", "FAILED_PRECONDITION":
		normalized.Retryable = false
		normalized.Temporary = false
	}
	return normalized
}

func normalizeGrokProviderError(resp *http.Response, body []byte) NormalizedProviderError {
	// xAI's public REST API follows the OpenAI-compatible error envelope for the
	// endpoints we proxy. Keeping a named entry point lets Grok diverge later
	// without leaking that compatibility assumption into registry callers.
	return normalizeOpenAIProviderError(resp, body)
}

func normalizeAntigravityProviderError(resp *http.Response, body []byte) NormalizedProviderError {
	// Antigravity is backed by Google's managed protocol and commonly surfaces
	// google.rpc-style status values. Keep it separate from Gemini at the
	// registry boundary so provider-specific overrides remain possible.
	return normalizeGeminiProviderError(resp, body)
}
