package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ProtectPeerAdminAuthIdentity prevents one administrator from attaching a
// new login identity to another administrator account. Shared Admin API Key
// requests are intentionally not treated as a personal administrator identity.
func (h *UserHandler) ProtectPeerAdminAuthIdentity(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		c.Abort()
		return
	}

	target, err := h.adminService.GetUser(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		c.Abort()
		return
	}

	if target.Role == service.RoleAdmin && getAdminIDFromContext(c) != target.ID {
		response.ErrorWithDetails(c, 403,
			"Cannot bind a login identity to another administrator",
			"ADMIN_PEER_AUTH_IDENTITY_PROTECTED", nil)
		c.Abort()
		return
	}

	c.Next()
}
