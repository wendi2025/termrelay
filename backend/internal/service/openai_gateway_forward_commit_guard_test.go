//go:build unit

package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestStopOpenAIResponsesFailoverAfterDownstreamCommit_NoCommitKeepsRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	failoverErr := &UpstreamFailoverError{StatusCode: http.StatusBadGateway}

	stopOpenAIResponsesFailoverAfterDownstreamCommit(c, failoverErr)

	require.True(t, failoverErr.ShouldRetryNextAccount())
	require.False(t, failoverErr.SafeToFailoverAfterWrite)
}

func TestStopOpenAIResponsesFailoverAfterDownstreamCommit_WrittenResponseStopsRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	failoverErr := &UpstreamFailoverError{StatusCode: http.StatusBadGateway}
	_, err := c.Writer.Write([]byte(": keepalive\n\n"))
	require.NoError(t, err)

	stopOpenAIResponsesFailoverAfterDownstreamCommit(c, failoverErr)

	require.False(t, failoverErr.ShouldRetryNextAccount())
	require.True(t, failoverErr.SafeToFailoverAfterWrite)
}

func TestStopOpenAIResponsesFailoverAfterDownstreamCommit_MarkedResponseStopsRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	failoverErr := &UpstreamFailoverError{StatusCode: http.StatusBadGateway}
	MarkResponseCommitted(c)

	stopOpenAIResponsesFailoverAfterDownstreamCommit(c, failoverErr)

	require.False(t, failoverErr.ShouldRetryNextAccount())
}
