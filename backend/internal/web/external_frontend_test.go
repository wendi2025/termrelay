//go:build embed

package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeExternalFrontendFixture(t *testing.T, root, marker string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "assets"), 0o755))
	html := `<!doctype html><html><head><title>fixture</title></head><body>` + marker + `<script type="module" src="/assets/app-12345678.js"></script></body></html>`
	require.NoError(t, os.WriteFile(filepath.Join(root, "index.html"), []byte(html), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "assets", "app-12345678.js"), []byte(`console.log("`+marker+`")`), 0o644))
}

func externalFrontendRouter(t *testing.T, root string) (*gin.Engine, *mockSettingsProvider) {
	t.Helper()
	provider := &mockSettingsProvider{settings: map[string]string{"site_name": "Smirel"}}
	server := newExternalFrontendServer(provider, root)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.CSPNonceKey, "test-nonce")
		c.Next()
	})
	router.Use(server.Middleware())
	router.NoRoute(func(c *gin.Context) {
		c.String(http.StatusTeapot, "embedded-fallback")
	})
	return router, provider
}

func TestExternalFrontendServer_FallsThroughWithoutBundle(t *testing.T) {
	root := t.TempDir()
	router, _ := externalFrontendRouter(t, root)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusTeapot, w.Code)
	assert.Equal(t, "embedded-fallback", w.Body.String())
}

func TestExternalFrontendServer_ServesSPAWithSettings(t *testing.T) {
	root := t.TempDir()
	writeExternalFrontendFixture(t, root, "release-one")
	router, provider := externalFrontendRouter(t, root)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/subscriptions", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	assert.Contains(t, w.Header().Get("Cache-Control"), "no-cache")
	assert.Contains(t, w.Body.String(), "release-one")
	assert.Contains(t, w.Body.String(), `window.__APP_CONFIG__=`)
	assert.Contains(t, w.Body.String(), `nonce="test-nonce"`)
	assert.Contains(t, w.Body.String(), "<title>Smirel - AI API Gateway</title>")
	assert.Equal(t, 1, provider.called)
}

func TestExternalFrontendServer_ServesFingerprintedAssets(t *testing.T) {
	root := t.TempDir()
	writeExternalFrontendFixture(t, root, "release-one")
	router, _ := externalFrontendRouter(t, root)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/assets/app-12345678.js", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "release-one")
	assert.Equal(t, staticAssetsCacheControl, w.Header().Get("Cache-Control"))
}

func TestExternalFrontendServer_BypassesAPI(t *testing.T) {
	root := t.TempDir()
	writeExternalFrontendFixture(t, root, "release-one")
	router, _ := externalFrontendRouter(t, root)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/public", nil))

	assert.Equal(t, http.StatusTeapot, w.Code)
	assert.Equal(t, "embedded-fallback", w.Body.String())
}

func TestExternalFrontendServer_DetectsHotSwapWithoutRestart(t *testing.T) {
	root := t.TempDir()
	writeExternalFrontendFixture(t, root, "release-one")
	provider := &mockSettingsProvider{settings: map[string]string{"site_name": "Smirel"}}
	server := newExternalFrontendServer(provider, root)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.CSPNonceKey, "test-nonce")
		c.Next()
	})
	router.Use(server.Middleware())

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Contains(t, first.Body.String(), "release-one")
	firstETag := first.Header().Get("ETag")
	require.NotEmpty(t, firstETag)

	writeExternalFrontendFixture(t, root, "release-two")
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, second.Code)
	assert.Contains(t, second.Body.String(), "release-two")
	assert.NotEqual(t, firstETag, second.Header().Get("ETag"))
	assert.GreaterOrEqual(t, provider.called, 2)
}

func TestCleanFrontendPathStaysRelative(t *testing.T) {
	for _, input := range []string{"/", "/assets/app.js", "/../index.html", "//assets/../index.html"} {
		cleaned := cleanFrontendPath(input)
		assert.False(t, filepath.IsAbs(cleaned), input)
		assert.False(t, strings.HasPrefix(cleaned, ".."), input)
	}
}
