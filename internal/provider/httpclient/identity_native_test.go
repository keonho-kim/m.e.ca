// Purpose: Verify native provenance, restricted proxy metadata and unsafe evidence failure paths.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestIdentityNative.
// Related: internal/provider/discovery/identify_ollama.go, internal/provider/discovery/identify_litellm.go.
package httpclient

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIdentityNativeProvenanceAndLimitedAccess(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.Ollama, connection.LiteLLM} {
		t.Run(string(kind), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if kind == connection.Ollama {
					fmt.Fprint(w, `{"model_info":{"general.source.repo_url":"https://huggingface.co/Qwen/Qwen3.8-27B","general.base_model.0.repo_url":"https://huggingface.co/Qwen/older-base"}}`)
					return
				}
				if r.URL.Path == "/model/info" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if r.URL.Path != "/model_group/info" || r.URL.Query().Get("model_group") != "alias/a+b" {
					t.Errorf("group query=%s", r.URL.String())
				}
				fmt.Fprint(w, `{"data":[{"model_group":"alias/a+b","providers":["vllm","ollama"]}]}`)
			}))
			defer server.Close()
			conn, err := connection.New(kind, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client, err := NewClient(conn, "")
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			report, err := catalogFor(client).Identify(context.Background(), "alias/a+b")
			if err != nil {
				t.Fatal(err)
			}
			if kind == connection.Ollama && (len(report.Originals()) != 1 || report.Originals()[0] != "https://huggingface.co/Qwen/Qwen3.8-27B") {
				t.Fatalf("provenance=%+v", report)
			}
			if kind == connection.LiteLLM && (!report.Unavailable || !report.Routed || len(report.Facts) != 2) {
				t.Fatalf("restricted report=%+v", report)
			}
		})
	}
}

func TestIdentityNativeRejectsUnsafeAndOversizedMetadata(t *testing.T) {
	for _, body := range []string{`{"data":[{"id":"alias","root":"Qwen/Qwen3.8-27B\u001b[31m"}]}`, `{"data":[{"id":"alias","root":"https://user:password@host/model?token=secret"}]}`, `not JSON`, strings.Repeat(" ", 4*1024*1024+1)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		conn, err := connection.New(connection.VLLM, server.URL)
		if err != nil {
			t.Fatal(err)
		}
		client, err := NewClient(conn, "secret")
		if err != nil {
			t.Fatal(err)
		}
		report, err := catalogFor(client).Identify(context.Background(), "alias")
		client.Close()
		server.Close()
		if err != nil || len(report.Originals()) != 0 || !report.Incomplete && !report.Unavailable {
			t.Fatalf("unsafe report=%+v err=%v", report, err)
		}
	}
}

func TestIdentityNativeRepositoryNamespaceIsNotProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"model_name":"alias","litellm_params":{"model":"Qwen/Qwen3.8-27B"}}]}`)
	}))
	defer server.Close()
	conn, err := connection.New(connection.LiteLLM, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client, err := NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	report, err := catalogFor(client).Identify(context.Background(), "alias")
	if err != nil {
		t.Fatal(err)
	}
	for _, fact := range report.Facts {
		if fact.Field == "provider" {
			t.Fatalf("repository namespace reported as provider: %+v", fact)
		}
	}
	if report.Originals()[0] != "Qwen/Qwen3.8-27B" {
		t.Fatal("repository identity rewritten")
	}
}
