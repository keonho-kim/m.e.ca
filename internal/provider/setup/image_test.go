// Purpose: Verify capability metadata, minimal probes and nonfatal unknown outcomes during connection setup.
// Pattern: Mock-provider integration tests.
// Usage: go test ./internal/provider/setup -run TestImage.
// Related: internal/provider/setup/image.go, internal/provider/diagnostic/image.go.
package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestImagePreparationMetadataAndProbe(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		kind                     connection.ProviderKind
		metadata, response, want string
		probes                   int32
	}{
		{"router-vision", connection.OpenRouter, `{"data":[{"id":"opaque","architecture":{"input_modalities":["text","image"]}}]}`, "", "supported", 0},
		{"router-unknown", connection.OpenRouter, `{"data":[{"id":"opaque"}]}`, "", "", 0},
		{"lm-vision", connection.LMStudio, `{"models":[{"key":"opaque","type":"llm","capabilities":{"vision":true}}]}`, "", "supported", 0},
		{"ollama-text", connection.Ollama, `{"capabilities":["completion"]}`, "", "unsupported", 0},
		{"sg-vision", connection.SGLang, `{"served_model_name":"opaque","has_image_understanding":true}`, "", "supported", 0},
		{"litellm-vision", connection.LiteLLM, `{"data":[{"model_name":"opaque","model_info":{"supports_vision":true}}]}`, "", "supported", 0},
		{"probe-accepted", connection.VLLM, `{"data":[{"id":"opaque"}]}`, `{"choices":[{"message":{"content":"anything"},"finish_reason":"stop"}]}`, "supported", 1},
		{"probe-rejected", connection.VLLM, `{"data":[{"id":"opaque"}]}`, `{"error":{"code":"image_input_not_supported","message":"no images"}}`, "unsupported", 1},
		{"probe-auth", connection.VLLM, `{"data":[{"id":"opaque"}]}`, "auth", "unknown", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var count atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/v1/chat/completions" {
					count.Add(1)
					var body struct {
						Model     string
						Tools     json.RawMessage
						MaxTokens int `json:"max_tokens"`
						Messages  []struct {
							Role    string
							Content []json.RawMessage
						}
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if body.Model != "opaque" || len(body.Messages) != 1 || body.Messages[0].Role != "user" || body.MaxTokens != 32 || len(body.Tools) > 0 {
						t.Errorf("invalid minimal probe: %+v", body)
					}
					if tc.response == "auth" {
						w.WriteHeader(http.StatusUnauthorized)
						fmt.Fprint(w, `{"error":{"message":"not authorized"}}`)
					} else {
						if tc.name == "probe-rejected" {
							w.WriteHeader(http.StatusBadRequest)
						}
						fmt.Fprint(w, tc.response)
					}
					return
				}
				if r.URL.Path == "/api/ps" {
					fmt.Fprint(w, `{"models":[]}`)
					return
				}
				if tc.kind == connection.SGLang && r.URL.Path == "/v1/models" {
					fmt.Fprint(w, `{"data":[{"id":"opaque"}]}`)
					return
				}
				fmt.Fprint(w, tc.metadata)
			}))
			defer server.Close()
			conn, err := connection.New(tc.kind, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := Prepare(context.Background(), conn, "", "opaque", Defaults{}, connection.Selection{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer prepared.Client.Close()
			if got := prepared.Selection.Details.ImageInput.State; got != tc.want || count.Load() != tc.probes {
				t.Fatalf("state=%q probes=%d; want %q/%d", got, count.Load(), tc.want, tc.probes)
			}
		})
	}
}

// Setup fixtures reject image probes independently of their ordinary response
// and Thinking assertions.
func rejectImageProbe(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
		return false
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if !bytes.Contains(body, []byte(`"image_url"`)) {
		return false
	}
	var request struct {
		MaxTokens int `json:"max_tokens"`
		Tools     json.RawMessage
	}
	if err := json.Unmarshal(body, &request); err != nil {
		t.Error(err)
	}
	if request.MaxTokens != 32 || len(request.Tools) > 0 {
		t.Error("capability check used a work request")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, `{"error":{"code":"image_input_not_supported","message":"fixture accepts text only"}}`)
	return true
}
