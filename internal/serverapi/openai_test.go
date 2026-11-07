// Purpose: Verify opt-in OpenAI routes use the existing engine and private HTTP boundary.
// Pattern: End-to-end HTTP contract tests with deterministic generation.
// Usage: go test ./internal/serverapi -run TestOpenAI.
// Related: internal/serverapi/openai/handler.go, internal/serverapi/security.go.
package serverapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func openAITransport(t *testing.T, f *fixture, protocol string) *Transport {
	t.Helper()
	transport, err := newTransport(t.Context(), f.api, protocol, testToken, []string{"localhost:8787"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	transport.enableOpenAI()
	t.Cleanup(transport.Close)
	return transport
}

func openAIRequest(t *testing.T, transport *Transport, method, path, body, key string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(deadline(t), method, "http://localhost:8787"+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	reply := httptest.NewRecorder()
	transport.ServeHTTP(reply, req)
	return reply
}

func TestOpenAIChatUsesEngineAndReturnsOriginalResult(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := openAITransport(t, f, "sse")
	reply := openAIRequest(t, transport, "POST", "/v1/chat/completions", `{"model":"meca","messages":[{"role":"user","content":"hello"}]}`, "chat-request")
	if reply.Code != http.StatusOK {
		t.Fatal(reply.Code, reply.Body.String())
	}
	var value struct {
		ID      string
		Choices []struct {
			Message      struct{ Content string }
			FinishReason string `json:"finish_reason"`
		}
	}
	if err := json.Unmarshal(reply.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.ID == "" || len(value.Choices) != 1 || value.Choices[0].Message.Content != "done" || value.Choices[0].FinishReason != "stop" {
		t.Fatal(reply.Body.String())
	}
	if f.app.State().Task.Status != "answered" {
		t.Fatal(f.app.State())
	}
}

func TestOpenAISocketModeStillRequiresBearer(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := openAITransport(t, f, "socket")
	req := httptest.NewRequest("GET", "http://localhost:8787/v1/models", nil)
	reply := httptest.NewRecorder()
	transport.ServeHTTP(reply, req)
	if reply.Code != http.StatusUnauthorized || !strings.Contains(reply.Body.String(), `"error"`) {
		t.Fatal(reply.Code, reply.Body.String())
	}
}
