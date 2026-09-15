package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// PlazaOfficialPricing 模型广场展示用的 LiteLLM 官方参考价（USD per token）。
// 字段为 nil 表示官方数据中该项缺失（0 视为未配置）。
type PlazaOfficialPricing struct {
	InputPrice        *float64
	OutputPrice       *float64
	CacheWritePrice   *float64 // 5m 缓存写入（= LiteLLM cache_creation）
	CacheWrite1hPrice *float64 // 1h 缓存写入（LiteLLM cache_creation_above_1hr）
	CacheReadPrice    *float64
}

// PlazaModel 模型广场中单个模型条目：用户侧模型名 + 渠道映射后的模型标识 + 渠道定价 + 官方参考价。
type PlazaModel struct {
	Name            string
	MappedModel     string
	Platform        string
	Pricing         *ChannelModelPricing
	OfficialPricing *PlazaOfficialPricing
}

// PlazaGroup 模型广场中以分组为顶层的条目。
//
// 与 AvailableGroupRef 相比多了 Description 与 Models；Models 来自该分组关联渠道的
// 支持模型。普通分组按平台隔离；Composite 分组保留具体传输平台，并在启用
// models_list_config 时按同一模型白名单约束展示，与真实推理权限保持一致。
type PlazaGroup struct {
	ID                 int64
	Name               string
	Description        string
	Platform           string
	SubscriptionType   string
	RateMultiplier     float64
	PeakRateEnabled    bool
	PeakStart          string
	PeakEnd            string
	PeakRateMultiplier float64
	IsExclusive        bool
	Models             []PlazaModel
}

// ListPlazaGroups 返回模型广场数据：每个活跃分组附带其可用模型与定价。
//
// 聚合口径与 ListAvailable 一致（Active 渠道、SupportedModels ∪ 全局定价回落、
// 平台隔离/Composite 逻辑归属），仅把顶层从渠道换成分组：
//   - 渠道按 lower(name) 排序后遍历，保证同名模型去重结果确定；
//   - 同分组同平台同名模型「先见者胜」，仅当已存条目无定价而新条目有定价时升级替换；
//   - Composite 分组可保留不同具体平台的同名模型；启用 models_list_config 时按白名单过滤；
//   - 每个模型附带 LiteLLM 官方参考价（查不到为 nil）；
//   - 只返回 Models 非空的分组；分组按 RateMultiplier 升序（同倍率按名称），
//     组内模型按名称、平台排序。
//
// 可见性过滤（专属分组）不在此层做，由 handler 按登录态裁剪。
func (s *ChannelService) ListPlazaGroups(ctx context.Context) ([]PlazaGroup, error) {
	channels, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}

	sort.SliceStable(channels, func(i, j int) bool {
		return strings.ToLower(channels[i].Name) < strings.ToLower(channels[j].Name)
	})

	byGroup := make(map[int64]*PlazaGroup, len(groups))
	groupConfig := make(map[int64]*Group, len(groups))
	order := make([]int64, 0, len(groups))
	for i := range groups {
		g := groups[i]
		groupConfig[g.ID] = &groups[i]
		byGroup[g.ID] = &PlazaGroup{
			ID:                 g.ID,
			Name:               g.Name,
			Description:        g.Description,
			Platform:           g.Platform,
			SubscriptionType:   g.SubscriptionType,
			RateMultiplier:     g.RateMultiplier,
			PeakRateEnabled:    g.PeakRateEnabled,
			PeakStart:          g.PeakStart,
			PeakEnd:            g.PeakEnd,
			PeakRateMultiplier: g.PeakRateMultiplier,
			IsExclusive:        g.IsExclusive,
		}
		order = append(order, g.ID)
	}

	type modelKey struct {
		platform string
		name     string
	}
	// Keep concrete platform in the key so a composite group can expose the
	// same public model name through more than one concrete transport without
	// silently collapsing one entry.
	modelIdx := make(map[int64]map[modelKey]int, len(groups))
	for i := range channels {
		ch := &channels[i]
		if ch.Status != StatusActive {
			continue
		}
		ch.normalizeBillingModelSource()
		supported := ch.SupportedModels()
		s.fillGlobalPricingFallback(supported)

		for _, gid := range ch.GroupIDs {
			pg, ok := byGroup[gid]
			if !ok {
				continue
			}
			idx := modelIdx[gid]
			if idx == nil {
				idx = make(map[modelKey]int, len(supported))
				modelIdx[gid] = idx
			}
			for j := range supported {
				m := supported[j]
				if !plazaGroupAllowsModel(groupConfig[gid], m) {
					continue
				}
				mappedModel := plazaMappedModel(ch, m.Platform, m.Name)
				key := modelKey{platform: m.Platform, name: m.Name}
				if at, seen := idx[key]; seen {
					// 先见者胜；仅当已存条目无定价而新条目有定价时升级。
					if pg.Models[at].Pricing == nil && m.Pricing != nil {
						pg.Models[at].Pricing = m.Pricing
						pg.Models[at].MappedModel = mappedModel
					}
					continue
				}
				idx[key] = len(pg.Models)
				pg.Models = append(pg.Models, PlazaModel{
					Name:        m.Name,
					MappedModel: mappedModel,
					Platform:    m.Platform,
					Pricing:     m.Pricing,
				})
			}
		}
	}

	officialMemo := make(map[string]*PlazaOfficialPricing)
	out := make([]PlazaGroup, 0, len(order))
	for _, gid := range order {
		pg := byGroup[gid]
		if len(pg.Models) == 0 {
			continue
		}
		sort.SliceStable(pg.Models, func(i, j int) bool {
			if pg.Models[i].Name != pg.Models[j].Name {
				return pg.Models[i].Name < pg.Models[j].Name
			}
			return pg.Models[i].Platform < pg.Models[j].Platform
		})
		for j := range pg.Models {
			pg.Models[j].OfficialPricing = s.lookupOfficialPricing(pg.Models[j].Name, officialMemo)
		}
		out = append(out, *pg)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RateMultiplier != out[j].RateMultiplier {
			return out[i].RateMultiplier < out[j].RateMultiplier
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// plazaGroupAllowsModel keeps logical model membership separate from the
// upstream transport protocol. Composite groups may intentionally expose
// Claude/Gemini/Grok/etc. through an OpenAI-compatible channel, so the group
// platform alone is not a sufficient ownership test. A custom model list is
// treated as the same allowlist used by the real inference path.
func plazaGroupAllowsModel(group *Group, model SupportedModel) bool {
	if group == nil {
		return false
	}
	if !isPlatformPricingMatch(group.Platform, model.Platform) {
		return false
	}
	return group.AllowsModel(model.Name)
}

// plazaMappedModel 返回 Model Plaza 可展示的渠道映射目标。
// Name 始终保留客户端应请求的模型名；仅在配置了实际改写时返回目标模型。
// 通配符目标无法可靠展开成单个实际标识，因此不在广场中伪造具体模型名。
func plazaMappedModel(ch *Channel, platform, model string) string {
	if ch == nil || model == "" {
		return ""
	}
	mapping := ch.ModelMapping[platform]
	if len(mapping) == 0 {
		return ""
	}

	// 运行时映射优先精确匹配，再匹配通配符。
	for src, target := range mapping {
		if !strings.EqualFold(src, model) {
			continue
		}
		if target == "" || strings.EqualFold(target, model) {
			return ""
		}
		if _, wildcard := splitWildcardSuffix(target); wildcard {
			return ""
		}
		return target
	}

	modelLower := strings.ToLower(model)
	for src, target := range mapping {
		prefix, wildcard := splitWildcardSuffix(src)
		if !wildcard || !strings.HasPrefix(modelLower, strings.ToLower(prefix)) {
			continue
		}
		if target == "" || strings.EqualFold(target, model) {
			return ""
		}
		if _, targetWildcard := splitWildcardSuffix(target); targetWildcard {
			return ""
		}
		return target
	}
	return ""
}

// lookupOfficialPricing 查询模型的 LiteLLM 官方参考价，带 memo 避免同名模型重复转换。
// pricingService 为 nil（测试场景）或查不到时返回 nil。
func (s *ChannelService) lookupOfficialPricing(modelName string, memo map[string]*PlazaOfficialPricing) *PlazaOfficialPricing {
	if s.pricingService == nil {
		return nil
	}
	if cached, ok := memo[modelName]; ok {
		return cached
	}
	var result *PlazaOfficialPricing
	if lp := s.pricingService.GetModelPricing(modelName); lp != nil && !lp.TokenPricingAbsent {
		result = &PlazaOfficialPricing{
			InputPrice:        nonZeroPtr(lp.InputCostPerToken),
			OutputPrice:       nonZeroPtr(lp.OutputCostPerToken),
			CacheWritePrice:   nonZeroPtr(lp.CacheCreationInputTokenCost),
			CacheWrite1hPrice: nonZeroPtr(lp.CacheCreationInputTokenCostAbove1hr),
			CacheReadPrice:    nonZeroPtr(lp.CacheReadInputTokenCost),
		}
		if result.InputPrice == nil && result.OutputPrice == nil &&
			result.CacheWritePrice == nil && result.CacheWrite1hPrice == nil && result.CacheReadPrice == nil {
			result = nil
		}
	}
	memo[modelName] = result
	return result
}
