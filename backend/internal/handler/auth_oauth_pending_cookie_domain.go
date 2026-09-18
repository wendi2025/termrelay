package handler

import (
	"net"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const oauthPendingCookieDomainEnv = "OAUTH_PENDING_COOKIE_DOMAIN"

// oauthPendingCookieDomain returns an explicitly configured parent domain for
// short-lived pending OAuth cookies. The value is ignored unless it is the
// current request host itself or one of its parent domains.
func oauthPendingCookieDomain(c *gin.Context) string {
	raw := strings.TrimSpace(os.Getenv(oauthPendingCookieDomainEnv))
	raw = strings.TrimPrefix(raw, ".")
	if raw == "" || strings.ContainsAny(raw, "/\\:") || net.ParseIP(raw) != nil {
		return ""
	}
	if c == nil || c.Request == nil {
		return ""
	}

	host := strings.TrimSpace(c.Request.Host)
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	domain := strings.TrimSuffix(strings.ToLower(raw), ".")
	if host == domain || strings.HasSuffix(host, "."+domain) {
		return domain
	}
	return ""
}
