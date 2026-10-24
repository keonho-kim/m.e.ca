// Purpose: Exercise known provider HTTP APIs using local mocks without an LLM.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient.
// Related: internal/provider/discovery/client.go, internal/provider/protocol/stream.go, internal/agent/model/model.go.

package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mockClient(t *testing.T, kind modelconn.ProviderKind, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	connection, err := modelconn.New(kind, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(connection, "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	return client
}

func TestListModelsProviderContracts(t *testing.T) {
	for _, test := range []struct {
		kind     modelconn.ProviderKind
		path     string
		response string
	}{
		{modelconn.Ollama, "/api/tags", `{"models":[{"name":"z-model"},{"name":"a-model"}]}`},
		{modelconn.VLLM, "/v1/models", `{"data":[{"id":"z-model"},{"id":"a-model"}]}`},
		{modelconn.LiteLLM, "/v1/models", `{"data":[{"id":"z-model"},{"id":"a-model"}]}`},
		{modelconn.SGLang, "/v1/models", `{"data":[{"id":"z-model"},{"id":"a-model"}]}`},
		{modelconn.LMStudio, "/api/v1/models", `{"models":[{"type":"llm","key":"z-model"},{"type":"embedding","key":"embedding"},{"type":"llm","key":"a-model"}]}`},
		{modelconn.OpenRouter, "/api/v1/models", `{"data":[{"id":"z-model"},{"id":"a-model"}]}`},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			client := mockClient(t, test.kind, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != test.path {
					t.Errorf("path = %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-secret" {
					t.Error("missing authentication")
				}
				fmt.Fprint(w, test.response)
			})
			models, err := catalogFor(client).ListModels(context.Background())
			if err != nil || len(models) != 2 || models[0].ID != "a-model" || models[1].ID != "z-model" {
				t.Fatalf("models = %+v, %v", models, err)
			}
		})
	}
}

func TestMalformedNativeCallsAreRejected(t *testing.T) {
	client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if _, exists := body["tools"]; exists {
			t.Error("native tool schema sent")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"function\":{\"name\":\"inspect_fixture\",\"arguments\":\"{}\"}}]}}]}\n\n")
	})
	err := client.Generate(context.Background(), modelapi.Request{ModelID: "chosen-model", MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "inspect"}}}, func(modelapi.Event) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "invalid tool call") {
		t.Fatalf("unexpected native call: %v", err)
	}
}

func TestOllamaStreamAndSequentialCapabilityProbe(t *testing.T) {
	client := mockClient(t, modelconn.Ollama, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ps" {
			fmt.Fprint(w, `{"models":[]}`)
			return
		}
		if r.URL.Path == "/api/show" {
			fmt.Fprint(w, `{"thinking":{"values":[true,false]}}`)
			return
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body struct {
			Messages []modelapi.Message `json:"messages"`
			Tools    json.RawMessage    `json:"tools"`
			Think    *bool              `json:"think"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Tools) != 0 {
			t.Error("native schemas were sent")
		}
		reply := "Hello!"
		w.Header().Set("Content-Type", "application/x-ndjson")
		thinking := ""
		if body.Think != nil && *body.Think {
			thinking = "brief"
		}
		encoded, err := json.Marshal(map[string]any{"message": map[string]string{"content": reply, "thinking": thinking}, "done": false})
		if err != nil {
			t.Error(err)
			return
		}
		fmt.Fprintln(w, string(encoded))
		fmt.Fprintln(w, `{"message":{"content":""},"done":true,"prompt_eval_count":12,"eval_count":5}`)
	})
	caps, err := diagnose(client, context.Background(), "model")
	if err != nil || !caps.Chat || !caps.Streaming {
		t.Fatalf("capabilities = %+v %v", caps, err)
	}
}

func TestGenerateStopsOnConsumerErrorAndRejectsTruncatedStreams(t *testing.T) {
	client := mockClient(t, modelconn.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"hello"}}]}`)
		fmt.Fprintln(w)
	})
	request := modelapi.Request{ModelID: "model", MaxOutput: 128, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}
	stop := errors.New("consumer stopped")
	err := client.Generate(context.Background(), request, func(modelapi.Event) error { return stop })
	if !errors.Is(err, stop) {
		t.Fatalf("consumer error = %v", err)
	}
	err = client.Generate(context.Background(), request, func(modelapi.Event) error { return nil })
	if err == nil {
		t.Fatal("truncated response reported success")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Generate(ctx, request, func(modelapi.Event) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
}

func TestProviderErrorsDoNotExposeAuthentication(t *testing.T) {
	client := mockClient(t, modelconn.VLLM, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "your credential is test-secret")
	})
	_, err := catalogFor(client).ListModels(context.Background())
	if err == nil || strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("unsafe error: %v", err)
	}
}

func TestSingleJSONResponseDoesNotClaimStreamingOrTools(t *testing.T) {
	client := mockClient(t, modelconn.OpenRouter, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}]}`)
	})
	capabilities, err := diagnose(client, context.Background(), "model")
	if err != nil || !capabilities.Chat || capabilities.Streaming {
		t.Fatalf("capabilities = %+v, %v", capabilities, err)
	}
}
