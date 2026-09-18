package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// ProviderAdapter is the stable protocol boundary between gateway orchestration
// and provider-specific behavior.
//
// Keep this interface intentionally small: transport, streaming, usage and
// model discovery are optional capabilities below. This prevents the adapter
// layer from becoming another god object as new providers/protocols are added.
type ProviderAdapter interface {
	Name() string
	Supports(account *Account) bool
	ResolveModel(account *Account, requestedModel string) (string, error)
	NormalizeError(resp *http.Response, body []byte) NormalizedProviderError
}

// ProviderProtocol is the client/wire protocol axis. Provider identity and
// protocol are intentionally orthogonal: one provider may serve several client
// protocols, and one protocol may be implemented by several providers.
type ProviderProtocol string

const (
	ProviderProtocolAnthropic       ProviderProtocol = "anthropic"
	ProviderProtocolChatCompletions ProviderProtocol = "chat_completions"
	ProviderProtocolResponses       ProviderProtocol = "responses"
	ProviderProtocolGemini          ProviderProtocol = "gemini"
)

// ProviderRequestInput contains only provider/request material. Scheduling,
// failover orchestration, response commit and billing remain gateway concerns.
type ProviderRequestInput struct {
	Account       *Account
	Protocol      ProviderProtocol
	Method        string
	Endpoint      string
	Model         string
	Body          []byte
	Stream        bool
	AuthToken     string
	ClientHeaders http.Header
}

// ProviderRequestPreparer resolves provider/account state that must be known
// before endpoint construction. It may return a copied input with ephemeral
// auth material, but must not perform scheduling, retries or billing.
type ProviderRequestPreparer interface {
	PrepareRequest(ctx context.Context, input ProviderRequestInput) (ProviderRequestInput, error)
}

// ProviderRequestBuilder builds the provider-native upstream request.
type ProviderRequestBuilder interface {
	BuildRequest(ctx context.Context, input ProviderRequestInput) (*http.Request, error)
}

// ProviderAuthApplier applies provider/account authentication after a request
// has been built. Implementations must never log or return raw secrets.
type ProviderAuthApplier interface {
	ApplyAuth(ctx context.Context, req *http.Request, input ProviderRequestInput) error
}

// ProviderRequestSender is optional. Normal HTTP providers should normally use
// the shared HTTPUpstream transport; only providers with a genuinely different
// transport contract (for example WebSocket) need to implement this capability.
type ProviderRequestSender interface {
	SendRequest(ctx context.Context, req *http.Request, account *Account) (*http.Response, error)
}

// ProviderStreamingParser parses provider-native streaming frames. It does not
// own client writes, flushing, cancellation, response commit or failover.
type ProviderStreamingParser interface {
	ParseStreaming(ctx context.Context, resp *http.Response, emit func(ProviderStreamEvent) error) (ProviderUsage, error)
}

// ProviderNonStreamingParser parses a provider-native non-streaming response.
type ProviderNonStreamingParser interface {
	ParseNonStreaming(ctx context.Context, resp *http.Response) (ProviderResponse, error)
}

// ProviderUsageExtractor extracts usage from a provider payload without
// performing billing side effects.
type ProviderUsageExtractor interface {
	ExtractUsage(payload []byte) (ProviderUsage, error)
}

// ProviderModelLister exposes a provider's current model list when the account
// shape supports a live listing endpoint.
type ProviderModelLister interface {
	ListModels(ctx context.Context, account *Account) ([]string, error)
}

// ProviderModelSyncer is reserved for providers whose sync process is more than
// a direct list call (for example a managed OAuth/manifest flow).
type ProviderModelSyncer interface {
	SyncModels(ctx context.Context, account *Account) ([]string, error)
}

// ProviderModelsRequestBuilder is the low-level capability used while existing
// model-sync code is migrated into adapters. It lets us remove platform switches
// without duplicating proven request/auth logic.
type ProviderModelsRequestBuilder interface {
	BuildModelsRequest(ctx context.Context, account *Account) (*http.Request, error)
}

// ProviderStreamEvent is intentionally transport-neutral. Raw may be retained
// during migration so existing wire-format writers can remain byte-compatible.
type ProviderStreamEvent struct {
	Type string
	Data []byte
	Raw  []byte
}

// ProviderUsage is the common usage envelope. Provider-specific counters stay
// in Extra until their billing semantics are explicitly normalized.
type ProviderUsage struct {
	InputTokens  int
	OutputTokens int
	CachedTokens int
	Extra        map[string]int64
}

// ProviderResponse is a provider-neutral non-streaming response envelope.
type ProviderResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Usage      ProviderUsage
}

// NormalizedProviderError is a side-effect-free error classification. Account
// cooldown/disable decisions belong to the scheduler/rate-limit layer.
type NormalizedProviderError struct {
	StatusCode int
	Code       string
	Type       string
	Message    string
	RequestID  string

	Retryable   bool
	Failover    bool
	RateLimited bool
	AuthFailure bool
	Temporary   bool
}

func (e NormalizedProviderError) Empty() bool {
	return e.StatusCode == 0 && e.Code == "" && e.Type == "" && e.Message == ""
}

// ProviderAdapterRegistry is concurrency-safe because request hot paths resolve
// adapters concurrently while registrations happen only during construction or
// tests. Duplicate names are rejected rather than silently replacing behavior.
type ProviderAdapterRegistry struct {
	mu       sync.RWMutex
	adapters []ProviderAdapter
	byName   map[string]ProviderAdapter
}

func NewProviderAdapterRegistry(adapters ...ProviderAdapter) (*ProviderAdapterRegistry, error) {
	r := &ProviderAdapterRegistry{byName: make(map[string]ProviderAdapter, len(adapters))}
	for _, adapter := range adapters {
		if err := r.Register(adapter); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func MustNewProviderAdapterRegistry(adapters ...ProviderAdapter) *ProviderAdapterRegistry {
	r, err := NewProviderAdapterRegistry(adapters...)
	if err != nil {
		panic(err)
	}
	return r
}

func (r *ProviderAdapterRegistry) Register(adapter ProviderAdapter) error {
	if r == nil {
		return errors.New("provider adapter registry is nil")
	}
	if adapter == nil {
		return errors.New("provider adapter is nil")
	}
	name := normalizeProviderAdapterName(adapter.Name())
	if name == "" {
		return errors.New("provider adapter name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byName == nil {
		r.byName = make(map[string]ProviderAdapter)
	}
	if _, exists := r.byName[name]; exists {
		return fmt.Errorf("provider adapter already registered: %s", name)
	}
	r.byName[name] = adapter
	r.adapters = append(r.adapters, adapter)
	return nil
}

func (r *ProviderAdapterRegistry) Get(name string) (ProviderAdapter, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.byName[normalizeProviderAdapterName(name)]
	return adapter, ok
}

func (r *ProviderAdapterRegistry) Resolve(account *Account) (ProviderAdapter, error) {
	if r == nil {
		return nil, errors.New("provider adapter registry is not configured")
	}
	if account == nil {
		return nil, errors.New("account is required")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched ProviderAdapter
	for _, adapter := range r.adapters {
		if !adapter.Supports(account) {
			continue
		}
		if matched != nil {
			return nil, fmt.Errorf(
				"ambiguous provider adapter match for platform=%s type=%s: %s, %s",
				account.Platform,
				account.Type,
				matched.Name(),
				adapter.Name(),
			)
		}
		matched = adapter
	}
	if matched != nil {
		return matched, nil
	}
	return nil, fmt.Errorf("no provider adapter for platform=%s type=%s", account.Platform, account.Type)
}

func (r *ProviderAdapterRegistry) Names() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		out = append(out, adapter.Name())
	}
	return out
}

func normalizeProviderAdapterName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeGenericProviderError(resp *http.Response, body []byte) NormalizedProviderError {
	if resp == nil {
		return NormalizedProviderError{}
	}

	status := resp.StatusCode
	message := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	requestID := strings.TrimSpace(resp.Header.Get("x-request-id"))
	if requestID == "" {
		requestID = strings.TrimSpace(resp.Header.Get("x-goog-request-id"))
	}
	if requestID == "" {
		requestID = strings.TrimSpace(resp.Header.Get("request-id"))
	}

	normalized := NormalizedProviderError{
		StatusCode: status,
		Code:       strings.TrimSpace(extractUpstreamErrorCode(body)),
		Message:    message,
		RequestID:  requestID,
	}

	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		normalized.Type = "authentication_error"
		normalized.AuthFailure = true
	case http.StatusTooManyRequests:
		normalized.Type = "rate_limit_error"
		normalized.RateLimited = true
		normalized.Retryable = true
		normalized.Failover = true
		normalized.Temporary = true
	case http.StatusRequestTimeout, http.StatusConflict, http.StatusTooEarly,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout, 529:
		normalized.Type = "upstream_error"
		normalized.Retryable = true
		normalized.Failover = true
		normalized.Temporary = true
	default:
		if status >= 500 {
			normalized.Type = "upstream_error"
			normalized.Retryable = true
			normalized.Failover = true
			normalized.Temporary = true
		} else if status >= 400 {
			normalized.Type = "invalid_request_error"
		}
	}

	return normalized
}
