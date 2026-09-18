package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// gatewayResponseCommitted reports whether any client-visible response state has
// been committed. Once this becomes true, gateway retry/failover must stop: the
// next upstream attempt could duplicate generation or splice two streams into a
// single client response even when the committed response contained no body yet.
func gatewayResponseCommitted(c *gin.Context) bool {
	if c == nil || c.Writer == nil {
		return false
	}
	if service.IsResponseCommitted(c) {
		return true
	}
	return c.Writer.Written()
}

func gatewayResponseIsSSE(c *gin.Context) bool {
	if c == nil || c.Writer == nil {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(c.Writer.Header().Get("Content-Type")))
	return strings.Contains(contentType, "text/event-stream")
}

// responsesStreamingAwareError preserves the Responses protocol after a wait
// ping or any other SSE output has committed HTTP 200. Once that happens JSON
// is no longer a valid fallback; emit a response.failed terminal event instead.
func (h *GatewayHandler) responsesStreamingAwareError(c *gin.Context, status int, code, message string, streamStarted bool) {
	if streamStarted || (gatewayResponseCommitted(c) && gatewayResponseIsSSE(c)) {
		service.MarkOpsStreamError(c, code, message, status)
		_ = writeResponsesFailedSSE(c, code, message)
		return
	}
	if gatewayResponseCommitted(c) {
		// A non-SSE response was already committed by a lower layer. Appending a
		// second JSON body would corrupt it, so the only safe action is to stop.
		return
	}
	h.responsesErrorResponse(c, status, code, message)
}
