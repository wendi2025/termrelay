package service

import (
	"fmt"
	"sort"
)

// ProviderCapability identifies an optional provider-adapter contract. The
// stable names are suitable for diagnostics/admin surfaces and deliberately do
// not expose concrete implementation types.
type ProviderCapability string

const (
	ProviderCapabilityRequestPreparer      ProviderCapability = "request_preparer"
	ProviderCapabilityRequestBuilder       ProviderCapability = "request_builder"
	ProviderCapabilityAuthApplier          ProviderCapability = "auth_applier"
	ProviderCapabilityRequestSender        ProviderCapability = "request_sender"
	ProviderCapabilityStreamingParser      ProviderCapability = "streaming_parser"
	ProviderCapabilityNonStreamingParser   ProviderCapability = "non_streaming_parser"
	ProviderCapabilityUsageExtractor       ProviderCapability = "usage_extractor"
	ProviderCapabilityModelLister          ProviderCapability = "model_lister"
	ProviderCapabilityModelSyncer          ProviderCapability = "model_syncer"
	ProviderCapabilityModelsRequestBuilder ProviderCapability = "models_request_builder"
)

// ProviderAdapterDescriptor is a read-only snapshot of an adapter's public
// extension surface. It contains no credentials, account data or functions.
type ProviderAdapterDescriptor struct {
	Name         string               `json:"name"`
	Capabilities []ProviderCapability `json:"capabilities"`
}

func detectProviderAdapterCapabilities(adapter ProviderAdapter) []ProviderCapability {
	if adapter == nil {
		return nil
	}

	capabilities := make([]ProviderCapability, 0, 10)
	if _, ok := adapter.(ProviderRequestPreparer); ok {
		capabilities = append(capabilities, ProviderCapabilityRequestPreparer)
	}
	if _, ok := adapter.(ProviderRequestBuilder); ok {
		capabilities = append(capabilities, ProviderCapabilityRequestBuilder)
	}
	if _, ok := adapter.(ProviderAuthApplier); ok {
		capabilities = append(capabilities, ProviderCapabilityAuthApplier)
	}
	if _, ok := adapter.(ProviderRequestSender); ok {
		capabilities = append(capabilities, ProviderCapabilityRequestSender)
	}
	if _, ok := adapter.(ProviderStreamingParser); ok {
		capabilities = append(capabilities, ProviderCapabilityStreamingParser)
	}
	if _, ok := adapter.(ProviderNonStreamingParser); ok {
		capabilities = append(capabilities, ProviderCapabilityNonStreamingParser)
	}
	if _, ok := adapter.(ProviderUsageExtractor); ok {
		capabilities = append(capabilities, ProviderCapabilityUsageExtractor)
	}
	if _, ok := adapter.(ProviderModelLister); ok {
		capabilities = append(capabilities, ProviderCapabilityModelLister)
	}
	if _, ok := adapter.(ProviderModelSyncer); ok {
		capabilities = append(capabilities, ProviderCapabilityModelSyncer)
	}
	if _, ok := adapter.(ProviderModelsRequestBuilder); ok {
		capabilities = append(capabilities, ProviderCapabilityModelsRequestBuilder)
	}
	return capabilities
}

// Descriptors returns a deterministic snapshot so callers can safely compare
// adapter topology across reloads and tests.
func (r *ProviderAdapterRegistry) Descriptors() []ProviderAdapterDescriptor {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	descriptors := make([]ProviderAdapterDescriptor, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		descriptors = append(descriptors, ProviderAdapterDescriptor{
			Name:         adapter.Name(),
			Capabilities: detectProviderAdapterCapabilities(adapter),
		})
	}
	r.mu.RUnlock()

	sort.Slice(descriptors, func(i, j int) bool {
		return normalizeProviderAdapterName(descriptors[i].Name) < normalizeProviderAdapterName(descriptors[j].Name)
	})
	return descriptors
}

// Capabilities resolves one adapter by registry name and returns a copy of its
// optional capability set.
func (r *ProviderAdapterRegistry) Capabilities(name string) ([]ProviderCapability, error) {
	adapter, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("provider adapter is not registered: %s", normalizeProviderAdapterName(name))
	}
	return detectProviderAdapterCapabilities(adapter), nil
}

// ResolveProviderCapability resolves an account and asserts one optional
// adapter contract in a single operation. It keeps capability checks uniform
// while preserving compile-time typing at call sites.
func ResolveProviderCapability[T any](registry *ProviderAdapterRegistry, account *Account) (T, error) {
	var zero T
	if registry == nil {
		return zero, fmt.Errorf("provider adapter registry is not configured")
	}
	adapter, err := registry.Resolve(account)
	if err != nil {
		return zero, err
	}
	capability, ok := any(adapter).(T)
	if !ok {
		return zero, fmt.Errorf("provider %s does not implement requested capability", adapter.Name())
	}
	return capability, nil
}
