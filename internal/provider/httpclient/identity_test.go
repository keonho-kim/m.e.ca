// Purpose: Verify provider metadata identifies selected aliases without exposing unrelated secrets.
// Pattern: HTTP adapter contract tests.
// Usage: go test ./internal/provider/httpclient -run TestIdentify.
// Related: internal/provider/discovery/identify.go, internal/provider/identity/report.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIdentifyProviderMetadata(t *testing.T) {
	cases := []struct {
		kind                        connection.ProviderKind
		path, body, state, evidence string
	}{
		{connection.VLLM, "/v1/models", `{"data":[{"id":"alias","root":"Qwen/Qwen3.8-27B"},{"id":"other","root":"secret"}]}`, "reported", "Qwen/Qwen3.8-27B"},
		{connection.SGLang, "/model_info", `{"served_model_name":"alias","model_path":"/weights/modelX","model_type":"qwen3","architectures":["Qwen3ForCausalLM"]}`, "reported", "qwen3"},
		{connection.Ollama, "/api/show", `{"details":{"family":"gemma4","parameter_size":"26B","quantization_level":"Q4_K_M"},"thinking":{"values":[false,true]},"system":"secret","modelfile":"secret"}`, "family", "Q4_K_M"},
		{connection.LMStudio, "/api/v1/models", `{"models":[{"type":"llm","key":"publisher/model-file","architecture":"qwen3","loaded_instances":[{"id":"alias"}],"capabilities":{"reasoning":{"allowed_options":["off","low","high"]}}}]}`, "reported", "publisher/model-file"},
		{connection.LiteLLM, "/model/info", `{"data":[{"model_name":"alias","litellm_params":{"model":"huggingface/Qwen/Qwen3.8-27B","api_key":"secret","api_base":"secret"}},{"model_name":"alias","litellm_params":{"model":"ollama/gemma4"}}]}`, "ambiguous", "gemma4"},
		{connection.OpenAICompatible, "/v1/models", `{"data":[{"id":"alias","owned_by":"team"}]}`, "unknown", "team"},
	}
	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					t.Errorf("path=%s", r.URL.Path)
				}
				if tc.kind == connection.Ollama {
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["model"] != "alias" {
						t.Errorf("show body=%v err=%v", body, err)
					}
				}
				fmt.Fprint(w, tc.body)
			}))
			t.Cleanup(server.Close)
			conn, err := connection.New(tc.kind, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client, err := NewClient(conn, "secret")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(client.Close)
			report, err := catalogFor(client).Identify(context.Background(), "alias")
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			if report.State() != tc.state || !strings.Contains(string(body), tc.evidence) || strings.Contains(string(body), "secret") {
				t.Fatalf("report=%s state=%s", body, report.State())
			}
		})
	}
}

func TestIdentifyFailureAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	conn, err := connection.New(connection.VLLM, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	report, err := catalogFor(client).Identify(context.Background(), "alias")
	if err != nil || !report.Unavailable || report.State() != "unknown" {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := catalogFor(client).Identify(ctx, "alias"); err != context.Canceled {
		t.Fatalf("cancel err=%v", err)
	}
}
