package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResponsesStreamingAwareError_WritesTerminalFailedEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Header("Content-Type", "text/event-stream")
	c.Writer.WriteHeaderNow()

	h := &GatewayHandler{}
	h.responsesStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "upstream failed", true)

	body := w.Body.String()
	require.Contains(t, body, "event: response.failed")
	require.Contains(t, body, `"status":"failed"`)
	require.Contains(t, body, `"code":"upstream_error"`)
	require.NotContains(t, body, `"type":"error"`)
}

func TestResponsesStreamingAwareError_UsesJSONBeforeCommit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	h := &GatewayHandler{}
	h.responsesStreamingAwareError(c, http.StatusBadGateway, "server_error", "failed", false)

	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), `"code":"server_error"`)
}

func TestHandleResponsesFailoverExhausted_StreamGetsTerminalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Header("Content-Type", "text/event-stream")
	c.Writer.WriteHeaderNow()

	h := &GatewayHandler{}
	h.handleResponsesFailoverExhausted(c, &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway}, true)

	require.Contains(t, w.Body.String(), "event: response.failed")
}
