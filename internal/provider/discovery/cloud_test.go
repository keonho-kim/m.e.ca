// Purpose: Check native model pagination, declared Claude controls and Gemini serving limits.
// Pattern: Local authenticated metadata contract tests.
// Usage: go test ./internal/provider/discovery -run TestCloud.
// Related: internal/provider/discovery/cloud.go, internal/provider/transport/transport.go.
package discovery

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/transport"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudCatalogsAndSelectedMetadata(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.Claude, connection.Gemini} {
		t.Run(string(kind), func(t *testing.T) {
			endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Has("key") {
					t.Error("API key appeared in query")
				}
				if kind == connection.Claude {
					if r.Header.Get("x-api-key") != "fixture-key" || r.Header.Get("anthropic-version") == "" {
						t.Error("Claude metadata authentication missing")
					}
					if r.URL.Path == "/v1/models/alias" {
						fmt.Fprint(w, `{"id":"resolved-id","max_input_tokens":200000,"max_tokens":16000,"capabilities":{"image_input":{"supported":true},"thinking":{"types":{"adaptive":{"supported":true},"disabled":{"supported":true}}},"effort":{"supported":true,"low":{"supported":true},"high":{"supported":true}}}}`)
						return
					}
					if r.URL.Query().Get("after_id") == "" {
						fmt.Fprint(w, `{"data":[{"id":"z-model"}],"has_more":true,"last_id":"z-model"}`)
					} else {
						fmt.Fprint(w, `{"data":[{"id":"a-model"}],"has_more":false}`)
					}
				} else {
					if r.Header.Get("x-goog-api-key") != "fixture-key" {
						t.Error("Gemini metadata authentication missing")
					}
					if r.URL.Path == "/v1beta/models/alias" {
						fmt.Fprint(w, `{"name":"models/resolved-id","inputTokenLimit":1000000,"outputTokenLimit":64000,"thinking":false,"supportedGenerationMethods":["generateContent"]}`)
						return
					}
					if r.URL.Query().Get("pageToken") == "" {
						fmt.Fprint(w, `{"models":[{"name":"models/z-model","supportedGenerationMethods":["generateContent"]},{"name":"models/embedding","supportedGenerationMethods":["embedContent"]}],"nextPageToken":"next"}`)
					} else {
						fmt.Fprint(w, `{"models":[{"name":"models/a-model","supportedGenerationMethods":["generateContent"]}]}`)
					}
				}
			}))
			defer endpoint.Close()
			conn, err := connection.New(kind, endpoint.URL)
			if err != nil {
				t.Fatal(err)
			}
			httpClient := transport.New("fixture-key", kind)
			defer httpClient.Close()
			client := Client{Connection: conn, HTTP: httpClient}
			models, err := client.ListModels(t.Context())
			if err != nil || len(models) != 2 || models[0].ID != "a-model" || models[1].ID != "z-model" {
				t.Fatal(models, err)
			}
			info, err := client.PrepareModel(t.Context(), "alias")
			if err != nil || info.ID != "alias" || info.ContextWindow == 0 || info.MaxOutput == 0 {
				t.Fatal(info, err)
			}
			if kind == connection.Claude && (!info.ImageInput.Supported() || info.Reasoning == nil || !info.Reasoning.On || !info.Reasoning.Off || len(info.Reasoning.Levels) != 2) {
				t.Fatal("Claude declarations lost", info)
			}
			if kind == connection.Gemini && (info.ThoughtSummary == nil || *info.ThoughtSummary) {
				t.Fatal("Gemini thinking capability lost")
			}
		})
	}
}

func TestCloudPaginationLoopIsRejected(t *testing.T) {
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"models":[],"nextPageToken":"repeated"}`)
	}))
	defer endpoint.Close()
	conn, err := connection.New(connection.Gemini, endpoint.URL)
	if err != nil {
		t.Fatal(err)
	}
	httpClient := transport.New("fixture", connection.Gemini)
	defer httpClient.Close()
	client := Client{Connection: conn, HTTP: httpClient}
	if _, err := client.ListModels(t.Context()); err == nil {
		t.Fatal("repeating provider cursor accepted")
	}
}
