package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestModelAccessAllowedUsesGroupAllowlist(t *testing.T) {
	apiKey := &service.APIKey{Group: &service.Group{
		Platform: service.PlatformOpenAI,
		ModelsListConfig: service.GroupModelsListConfig{
			Enabled: true,
			Models:  []string{"cqu-default", "gpt-*"},
		},
	}}

	if !modelAccessAllowed(apiKey, "cqu-default") {
		t.Fatal("expected cqu-default to be allowed")
	}
	if !modelAccessAllowed(apiKey, "gpt-5.6") {
		t.Fatal("expected gpt-* wildcard to allow gpt-5.6")
	}
	if modelAccessAllowed(apiKey, "MiniMax-M3") {
		t.Fatal("expected MiniMax-M3 to be denied")
	}
}

func TestCompositeAdmissionRejectsHiddenModelForNonCompositeGroup(t *testing.T) {
	apiKey := &service.APIKey{Group: &service.Group{
		Platform: service.PlatformOpenAI,
		ModelsListConfig: service.GroupModelsListConfig{
			Enabled: true,
			Models:  []string{"cqu-default"},
		},
	}}

	if compositeTargetPlatformAllowed(nil, apiKey, "MiniMax-M3", service.PlatformOpenAI) {
		t.Fatal("hidden model must be rejected before OpenAI-compatible routing")
	}
	if compositeTargetPlatformResolved(nil, apiKey, "MiniMax-M3") {
		t.Fatal("hidden model must be rejected before generic gateway routing")
	}
	if !compositeTargetPlatformAllowed(nil, apiKey, "cqu-default", service.PlatformOpenAI) {
		t.Fatal("allowed model should continue through routing")
	}
	if !compositeTargetPlatformResolved(nil, apiKey, "cqu-default") {
		t.Fatal("allowed model should continue through generic routing")
	}
}
