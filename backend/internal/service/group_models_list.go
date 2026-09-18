package service

import "strings"

func normalizeGroupModelsListConfig(cfg GroupModelsListConfig) GroupModelsListConfig {
	out := GroupModelsListConfig{Enabled: cfg.Enabled}
	if len(cfg.Models) == 0 {
		return out
	}

	seen := make(map[string]struct{}, len(cfg.Models))
	out.Models = make([]string, 0, len(cfg.Models))
	for _, model := range cfg.Models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		out.Models = append(out.Models, model)
	}
	if len(out.Models) == 0 {
		out.Models = nil
	}
	return out
}

func (g *Group) CustomModelsListEnabled() bool {
	return g != nil && g.ModelsListConfig.Enabled && len(g.ModelsListConfig.Models) > 0
}

// AllowsModel reports whether a client authenticated into this group may use
// the requested public model. The same models_list_config that shapes
// /v1/models is treated as an allowlist on real inference requests so a client
// cannot bypass the UI by manually constructing a request for a hidden model.
//
// Existing groups remain backwards-compatible: when the custom list is not
// enabled (or contains no usable entries), model access is unrestricted and
// the normal account/model routing rules decide availability.
func (g *Group) AllowsModel(model string) bool {
	if !g.CustomModelsListEnabled() {
		return true
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	for _, pattern := range g.ModelsListConfig.Models {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern == model {
			return true
		}
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(model, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}
