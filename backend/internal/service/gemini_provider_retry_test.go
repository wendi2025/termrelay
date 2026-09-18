package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type geminiSequentialHTTPUpstreamStub struct {
	responses     []*http.Response
	requestBodies [][]byte
	calls         int
}

func (s *geminiSequentialHTTPUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	s.requestBodies = append(s.requestBodies, append([]byte(nil), body...))

	if s.calls >= len(s.responses) {
		return nil, fmt.Errorf("unexpected upstream call %d", s.calls+1)
	}
	resp := *s.responses[s.calls]
	s.calls++
	return &resp, nil
}

func (s *geminiSequentialHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestGeminiMessagesCompatSignatureRetryRebuildsProviderRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	httpStub := &geminiSequentialHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"gemini-signature-1"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"thought_signature is required"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"gemini-success-2"}},
				Body: io.NopCloser(strings.NewReader(
					`{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":1}}`,
				)),
			},
		},
	}

	svc := &GeminiMessagesCompatService{
		httpUpstream: httpStub,
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       901,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "gemini-api-key",
			"model_mapping": map[string]any{
				"claude-sonnet-4-5": "gemini-2.5-flash",
			},
		},
		Concurrency: 1,
	}

	body := []byte(`{
		"model":"claude-sonnet-4-5",
		"max_tokens":64,
		"thinking":{"type":"enabled","budget_tokens":1024},
		"messages":[
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"private-thought","signature":"old-signature"},
				{"type":"text","text":"previous answer"}
			]},
			{"role":"user","content":"continue"}
		]
	}`)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))

	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, httpStub.calls)
	require.Len(t, httpStub.requestBodies, 2)
	require.NotEqual(t, string(httpStub.requestBodies[0]), string(httpStub.requestBodies[1]), "signature fallback must rebuild the provider request with the downgraded payload")
	require.Contains(t, string(httpStub.requestBodies[0]), "private-thought")
	require.Contains(t, string(httpStub.requestBodies[1]), `"text":"private-thought"`, "thinking content must be downgraded to ordinary text")
	require.NotContains(t, string(httpStub.requestBodies[1]), "thinkingConfig", "signature fallback must disable Gemini thinking")
	require.Equal(t, "gemini-2.5-flash", result.UpstreamModel)
}
