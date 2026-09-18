package service

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"unicode"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	accountPoolProviderIDExtraKey = "provider_id"
	maxAccountPoolProviderIDLen   = 128
)

// AccountPoolProviderIdentity is the scheduler-facing provider identity. It is
// deliberately separate from Account: one provider may own many API-key/OAuth
// accounts, while credentials and per-account health remain account-scoped.
type AccountPoolProviderIdentity struct {
	Key      string `json:"key"`
	Platform string `json:"platform"`
	Origin   string `json:"origin,omitempty"`
}

// ExplicitProviderID returns the normalized operator-owned logical provider ID.
// Empty means the account falls back to the derived platform+origin identity.
func (a *Account) ExplicitProviderID() string {
	if a == nil {
		return ""
	}
	return strings.TrimSpace(a.GetExtraString(accountPoolProviderIDExtraKey))
}

// NormalizeAccountPoolProviderID validates the optional logical provider ID. A
// typed field wins over the legacy Extra value. Empty explicitly clears the
// logical provider by storing JSON null, which makes readers use derived identity
// while remaining safe for key-level JSONB merge updates.
func NormalizeAccountPoolProviderID(extra map[string]any, explicit *string) (map[string]any, error) {
	raw, hasProviderID := any(nil), false
	if extra != nil {
		raw, hasProviderID = extra[accountPoolProviderIDExtraKey]
	}
	if explicit != nil {
		raw = *explicit
		hasProviderID = true
	}
	if !hasProviderID {
		return extra, nil
	}

	providerID := ""
	if raw != nil {
		value, ok := raw.(string)
		if !ok {
			return nil, infraerrors.BadRequest(
				"INVALID_ACCOUNT_PROVIDER_ID",
				"provider_id must be a string",
			)
		}
		providerID = strings.TrimSpace(value)
	}
	if len(providerID) > maxAccountPoolProviderIDLen {
		return nil, infraerrors.BadRequest(
			"INVALID_ACCOUNT_PROVIDER_ID",
			"provider_id must be at most 128 characters",
		)
	}
	for _, r := range providerID {
		if unicode.IsControl(r) {
			return nil, infraerrors.BadRequest(
				"INVALID_ACCOUNT_PROVIDER_ID",
				"provider_id must not contain control characters",
			)
		}
	}

	normalized := make(map[string]any, len(extra)+1)
	for key, value := range extra {
		normalized[key] = value
	}
	if providerID == "" {
		normalized[accountPoolProviderIDExtraKey] = nil
	} else {
		normalized[accountPoolProviderIDExtraKey] = providerID
	}
	return normalized, nil
}

// ProviderIdentity derives a stable non-secret provider identity. Operators can
// set Extra["provider_id"] for an explicit logical provider. Otherwise API-key
// accounts sharing the same platform + upstream origin are grouped together.
// API keys, OAuth tokens and other credential material never participate in the
// identity or telemetry key.
func (a *Account) ProviderIdentity() AccountPoolProviderIdentity {
	if a == nil {
		return AccountPoolProviderIdentity{}
	}
	platform := strings.ToLower(strings.TrimSpace(a.Platform))
	if explicit := a.ExplicitProviderID(); explicit != "" {
		return AccountPoolProviderIdentity{
			Key:      "provider:" + explicit,
			Platform: platform,
		}
	}

	origin := normalizeAccountPoolProviderOrigin(a.GetCredential("base_url"))
	material := platform
	if origin != "" {
		material += "|" + origin
	}
	digest := sha256.Sum256([]byte(material))
	return AccountPoolProviderIdentity{
		Key:      "derived:" + hex.EncodeToString(digest[:8]),
		Platform: platform,
		Origin:   origin,
	}
}

func normalizeAccountPoolProviderOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(raw, "/")
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String()
}
