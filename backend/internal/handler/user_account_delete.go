package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

type deleteOwnAccountRequest struct {
	Confirmation string `json:"confirmation"`
}

// DeleteOwnAccount DELETE /user/account
//
// 注销当前登录用户。该操作使用 User 的软删除语义，保留必要的计费/审计历史，
// 同时使该用户及其 API Key 无法继续通过鉴权。为避免误操作，客户端必须回传
// 当前账户邮箱作为确认文本。管理员账户不得通过用户侧入口自行注销。
func (h *UserHandler) DeleteOwnAccount(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req deleteOwnAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入当前账户邮箱以确认注销")
		return
	}

	user, err := h.userService.GetProfile(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
		response.BadRequest(c, "管理员账户不能直接注销，请先转移管理员权限")
		return
	}

	if !strings.EqualFold(strings.TrimSpace(req.Confirmation), strings.TrimSpace(user.Email)) {
		response.BadRequest(c, "确认邮箱与当前账户不一致")
		return
	}

	if err := h.userService.Delete(c.Request.Context(), subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 用户已经注销后，尽力清理所有 refresh session。即使 Redis 清理暂时失败，
	// 软删除用户本身也会使后续鉴权无法解析到有效用户。
	if h.authService != nil {
		_ = h.authService.RevokeAllUserSessions(c.Request.Context(), subject.UserID)
	}

	response.Success(c, gin.H{"deleted": true})
}
