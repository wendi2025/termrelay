// Package dto provides data transfer objects for HTTP handlers.
package dto

import (
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// RedactCredentials 复制一份 in，剥离 service.SensitiveCredentialKeys 列出的所有敏感子键，
// 并产出一个 has_<key> 状态 map 表示哪些敏感键存在且非零值。
//
// 输入 nil 时返回 nil, nil（避免响应里出现空对象）。
// 不修改入参；调用方拿到的 out 可安全序列化进 JSON 返回前端。
func RedactCredentials(in map[string]any) (out map[string]any, status map[string]bool) {
	if in == nil {
		return nil, nil
	}
	out = make(map[string]any, len(in))
	for k, v := range in {
		if service.IsSensitiveCredentialKey(k) {
			if isCredentialValuePresent(v) {
				if status == nil {
					status = make(map[string]bool, 4)
				}
				status["has_"+k] = true
			}
			continue
		}
		out[k] = redactCredentialDisplayValue(k, v)
	}
	return out, status
}

// redactCredentialDisplayValue sanitizes credential metadata that is useful to
// administrators but may itself contain embedded secrets. In particular,
// base_url may contain URL userinfo, query parameters, or fragments supplied by
// an upstream gateway. Those components must never be serialized back to the
// admin API. Invalid/non-absolute values fail closed instead of returning the
// original string.
func redactCredentialDisplayValue(key string, value any) any {
	if !strings.EqualFold(strings.TrimSpace(key), "base_url") {
		return value
	}
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.RawFragment = ""
	return parsed.String()
}

// isCredentialValuePresent 判断值是否"存在且非零"。空字符串、nil、false 均视为未配置；
// 其余非零类型（数字、对象、字符串等）视为已配置。
func isCredentialValuePresent(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return x != ""
	case bool:
		return x
	default:
		return true
	}
}
