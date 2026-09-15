//go:build unit

package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func runAuthIdentityGuardRequest(t *testing.T, target service.User, actorID int64, authMethod string) (int, string, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	adminSvc.users = []service.User{target}
	h := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)

	called := false
	router := gin.New()
	router.POST("/users/:id/auth-identities",
		func(c *gin.Context) {
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: actorID})
			c.Set("auth_method", authMethod)
			c.Next()
		},
		h.ProtectPeerAdminAuthIdentity,
		func(c *gin.Context) {
			called = true
			c.Status(http.StatusNoContent)
		},
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/1/auth-identities", nil)
	router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String(), called
}

func TestProtectPeerAdminAuthIdentityRejectsPeerJWTAdmin(t *testing.T) {
	code, body, called := runAuthIdentityGuardRequest(t, service.User{ID: 1, Role: service.RoleAdmin, Status: service.StatusActive}, 2, "jwt")
	require.Equal(t, http.StatusForbidden, code)
	require.Contains(t, body, "ADMIN_PEER_AUTH_IDENTITY_PROTECTED")
	require.False(t, called)
}

func TestProtectPeerAdminAuthIdentityAllowsSelfJWTAdmin(t *testing.T) {
	code, _, called := runAuthIdentityGuardRequest(t, service.User{ID: 1, Role: service.RoleAdmin, Status: service.StatusActive}, 1, "jwt")
	require.Equal(t, http.StatusNoContent, code)
	require.True(t, called)
}

func TestProtectPeerAdminAuthIdentityRejectsSharedAdminAPIKey(t *testing.T) {
	code, body, called := runAuthIdentityGuardRequest(t, service.User{ID: 1, Role: service.RoleAdmin, Status: service.StatusActive}, 1, "admin_api_key")
	require.Equal(t, http.StatusForbidden, code)
	require.Contains(t, body, "ADMIN_PEER_AUTH_IDENTITY_PROTECTED")
	require.False(t, called)
}

func TestProtectPeerAdminAuthIdentityAllowsOrdinaryUserRepair(t *testing.T) {
	code, _, called := runAuthIdentityGuardRequest(t, service.User{ID: 1, Role: service.RoleUser, Status: service.StatusActive}, 2, "jwt")
	require.Equal(t, http.StatusNoContent, code)
	require.True(t, called)
}
