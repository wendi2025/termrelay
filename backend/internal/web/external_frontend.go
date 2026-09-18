//go:build embed

package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// ExternalFrontendServer serves a complete frontend bundle from data/frontend.
// When data/frontend/index.html is absent, requests fall through to the embedded
// frontend. The external root is deliberately separate from data/public, which
// remains the legacy per-file override directory.
type ExternalFrontendServer struct {
	root     string
	cache    *HTMLCache
	settings PublicSettingsProvider
}

// NewExternalFrontendServer creates the runtime frontend bundle server.
func NewExternalFrontendServer(settingsProvider PublicSettingsProvider) *ExternalFrontendServer {
	return newExternalFrontendServer(settingsProvider, filepath.Join("data", "frontend"))
}

func newExternalFrontendServer(settingsProvider PublicSettingsProvider, root string) *ExternalFrontendServer {
	return &ExternalFrontendServer{
		root:     root,
		cache:    NewHTMLCache(),
		settings: settingsProvider,
	}
}

// InvalidateCache invalidates rendered external index HTML after public settings change.
func (s *ExternalFrontendServer) InvalidateCache() {
	if s != nil && s.cache != nil {
		s.cache.Invalidate()
	}
}

// Middleware serves the external frontend only when a complete bundle is active.
// API routes always pass through untouched.
func (s *ExternalFrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s == nil || shouldBypassEmbeddedFrontend(c.Request.URL.Path) || !s.bundleActive() {
			c.Next()
			return
		}

		cleanPath := cleanFrontendPath(c.Request.URL.Path)
		if cleanPath != "index.html" && s.tryServeFile(c, cleanPath) {
			return
		}

		// Any non-file frontend route is an SPA route and falls back to the
		// external index. If the index disappears during an atomic deployment,
		// serveIndexHTML returns false and we safely fall through to the embedded UI.
		if s.serveIndexHTML(c) {
			return
		}
		c.Next()
	}
}

func (s *ExternalFrontendServer) bundleActive() bool {
	if s == nil || s.root == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(s.root, "index.html"))
	return err == nil && !info.IsDir() && info.Size() > 0
}

func cleanFrontendPath(requestPath string) string {
	clean := filepath.Clean("/" + strings.TrimSpace(requestPath))
	clean = strings.TrimPrefix(clean, string(filepath.Separator))
	if clean == "" || clean == "." {
		return "index.html"
	}
	return clean
}

func (s *ExternalFrontendServer) tryServeFile(c *gin.Context, cleanPath string) bool {
	filePath := filepath.Join(s.root, cleanFrontendPath(cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}

	applyStaticAssetCacheHeaders(c.Writer.Header(), cleanPath)
	c.File(filePath)
	c.Abort()
	return true
}

func (s *ExternalFrontendServer) serveIndexHTML(c *gin.Context) bool {
	baseHTML, err := os.ReadFile(filepath.Join(s.root, "index.html"))
	if err != nil || len(baseHTML) == 0 {
		return false
	}

	// SetBaseHTML invalidates a cached render only when the deployed index hash
	// changes, so a symlink release switch becomes visible without a process restart.
	s.cache.SetBaseHTML(baseHTML)
	nonce := middleware.GetNonceFromContext(c)

	if cached := s.cache.Get(); cached != nil {
		if match := c.GetHeader("If-None-Match"); match == cached.ETag {
			c.Status(http.StatusNotModified)
			c.Abort()
			return true
		}
		content := replaceNoncePlaceholder(cached.Content, nonce)
		c.Header("ETag", cached.ETag)
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
		c.Abort()
		return true
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	settings, err := s.settings.GetPublicSettingsForInjection(ctx)
	if err != nil {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", baseHTML)
		c.Abort()
		return true
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", baseHTML)
		c.Abort()
		return true
	}

	rendered := injectSettingsIntoHTML(baseHTML, settingsJSON)
	s.cache.Set(rendered, settingsJSON)
	content := replaceNoncePlaceholder(rendered, nonce)
	if cached := s.cache.Get(); cached != nil {
		c.Header("ETag", cached.ETag)
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
	return true
}

func injectSettingsIntoHTML(baseHTML, settingsJSON []byte) []byte {
	script := []byte(`<script nonce="` + NonceHTMLPlaceholder + `">window.__APP_CONFIG__=` + string(settingsJSON) + `;</script>`)
	headClose := []byte("</head>")
	result := bytes.Replace(baseHTML, headClose, append(script, headClose...), 1)
	result = injectSiteTitle(result, settingsJSON)
	result = injectSiteFavicon(result, settingsJSON)
	return result
}
