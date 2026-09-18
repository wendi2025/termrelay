package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newOAuthCookieTestContext(host string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "https://"+host+"/api/v1/auth/oauth/google/callback", nil)
	return ctx, recorder
}

func TestOAuthPendingCookieDomainAllowsConfiguredParentDomain(t *testing.T) {
	t.Setenv(oauthPendingCookieDomainEnv, "smirel.com")
	ctx, _ := newOAuthCookieTestContext("api.smirel.com")
	if got := oauthPendingCookieDomain(ctx); got != "smirel.com" {
		t.Fatalf("oauthPendingCookieDomain() = %q, want smirel.com", got)
	}
}

func TestOAuthPendingCookieDomainRejectsUnrelatedDomain(t *testing.T) {
	t.Setenv(oauthPendingCookieDomainEnv, "example.com")
	ctx, _ := newOAuthCookieTestContext("api.smirel.com")
	if got := oauthPendingCookieDomain(ctx); got != "" {
		t.Fatalf("oauthPendingCookieDomain() = %q, want empty", got)
	}
}

func TestOAuthPendingCookiesCarrySharedDomain(t *testing.T) {
	t.Setenv(oauthPendingCookieDomainEnv, ".smirel.com")
	ctx, recorder := newOAuthCookieTestContext("api.smirel.com")
	setOAuthPendingSessionCookie(ctx, "session-token", true)
	setOAuthPendingBrowserCookie(ctx, "browser-token", true)

	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want 2", len(cookies))
	}
	for _, cookie := range cookies {
		if cookie.Domain != "smirel.com" {
			t.Fatalf("cookie %s domain = %q, want smirel.com", cookie.Name, cookie.Domain)
		}
		if !cookie.Secure || !cookie.HttpOnly {
			t.Fatalf("cookie %s missing Secure/HttpOnly flags", cookie.Name)
		}
		if cookie.Path != oauthPendingSessionCookiePath && cookie.Path != oauthPendingBrowserCookiePath {
			t.Fatalf("cookie %s unexpected path %q", cookie.Name, cookie.Path)
		}
		if strings.TrimSpace(cookie.Value) == "" {
			t.Fatalf("cookie %s has empty value", cookie.Name)
		}
	}
}
