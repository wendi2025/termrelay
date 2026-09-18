package service

import "testing"

func TestProviderIdentityGroupsAccountsByOriginWithoutSecrets(t *testing.T) {
	first := &Account{
		Platform: PlatformOpenAI,
		Credentials: map[string]any{
			"base_url": "https://relay.example.com/v1/",
			"api_key":  "secret-one",
		},
	}
	second := &Account{
		Platform: PlatformOpenAI,
		Credentials: map[string]any{
			"base_url": "https://RELAY.EXAMPLE.COM/v1",
			"api_key":  "secret-two",
		},
	}

	left := first.ProviderIdentity()
	right := second.ProviderIdentity()
	if left.Key == "" || left.Key != right.Key {
		t.Fatalf("same provider origin should share identity: left=%+v right=%+v", left, right)
	}
	if left.Origin != "https://relay.example.com/v1" {
		t.Fatalf("unexpected normalized origin: %q", left.Origin)
	}
	if left.Key == "secret-one" || right.Key == "secret-two" {
		t.Fatal("provider identity must never contain API keys")
	}
}

func TestProviderIdentitySeparatesDifferentOrigins(t *testing.T) {
	first := (&Account{Platform: PlatformOpenAI, Credentials: map[string]any{"base_url": "https://a.example.com/v1"}}).ProviderIdentity()
	second := (&Account{Platform: PlatformOpenAI, Credentials: map[string]any{"base_url": "https://b.example.com/v1"}}).ProviderIdentity()
	if first.Key == second.Key {
		t.Fatalf("different provider origins must not share identity: %+v %+v", first, second)
	}
}

func TestProviderIdentityExplicitProviderIDOverridesOrigin(t *testing.T) {
	first := (&Account{
		Platform: PlatformOpenAI,
		Extra:    map[string]any{accountPoolProviderIDExtraKey: "llmgw"},
		Credentials: map[string]any{
			"base_url": "https://one.example.com/v1",
		},
	}).ProviderIdentity()
	second := (&Account{
		Platform: PlatformOpenAI,
		Extra:    map[string]any{accountPoolProviderIDExtraKey: "llmgw"},
		Credentials: map[string]any{
			"base_url": "https://two.example.com/v1",
		},
	}).ProviderIdentity()
	if first.Key != "provider:llmgw" || second.Key != first.Key {
		t.Fatalf("explicit provider id should group accounts: first=%+v second=%+v", first, second)
	}
}

func TestNormalizeAccountPoolProviderID(t *testing.T) {
	t.Run("explicit typed value wins and trims", func(t *testing.T) {
		providerID := "  llmgw-primary  "
		extra, err := NormalizeAccountPoolProviderID(map[string]any{
			accountPoolProviderIDExtraKey: "legacy",
			"keep":                        true,
		}, &providerID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := extra[accountPoolProviderIDExtraKey]; got != "llmgw-primary" {
			t.Fatalf("provider_id=%v want=llmgw-primary", got)
		}
		if extra["keep"] != true {
			t.Fatal("unrelated extra field was not preserved")
		}
	})

	t.Run("empty explicit value clears to derived mode", func(t *testing.T) {
		providerID := ""
		extra, err := NormalizeAccountPoolProviderID(nil, &providerID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value, ok := extra[accountPoolProviderIDExtraKey]; !ok || value != nil {
			t.Fatalf("expected provider_id JSON null, got %#v", extra)
		}
	})

	t.Run("rejects malformed raw value", func(t *testing.T) {
		if _, err := NormalizeAccountPoolProviderID(map[string]any{accountPoolProviderIDExtraKey: 42}, nil); err == nil {
			t.Fatal("expected non-string provider_id to be rejected")
		}
	})
}
