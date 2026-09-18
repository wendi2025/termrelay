//go:build !embed

package web

import "github.com/gin-gonic/gin"

// ExternalFrontendServer is a no-op in non-embed builds. Runtime frontend
// overrides are a production/embed feature; development keeps using Vite.
type ExternalFrontendServer struct{}

func NewExternalFrontendServer(settingsProvider PublicSettingsProvider) *ExternalFrontendServer {
	return &ExternalFrontendServer{}
}

func (s *ExternalFrontendServer) InvalidateCache() {}

func (s *ExternalFrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
