//go:build unit

package admin

import (
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetAdminIDFromContextRejectsSharedAdminAPIKeyIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	c.Set("auth_method", "admin_api_key")

	require.Equal(t, int64(0), getAdminIDFromContext(c))
}

func TestGetAdminIDFromContextKeepsJWTAdminIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	c.Set("auth_method", "jwt")

	require.Equal(t, int64(42), getAdminIDFromContext(c))
}
