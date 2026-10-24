// Purpose: Verify native model metadata preserves explicit default ON/OFF values.
// Pattern: Local HTTP metadata contract tests.
// Usage: go test ./internal/provider/httpclient -run TestAdvertisedThinkingDefault.
// Related: internal/provider/discovery/ollama_settings.go, internal/provider/discovery/lmstudio_settings.go, internal/provider/discovery/openrouter.go.
package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestAdvertisedThinkingDefault(t *testing.T) {
	for _, test := range []struct {
		kind connection.ProviderKind
		body string
	}{
		{connection.Ollama, `{"thinking":{"values":[true,false],"default":false}}`},
		{connection.LMStudio, `{"models":[{"type":"llm","key":"m","capabilities":{"reasoning":{"allowed_options":["off","low","high"],"default":"off"}}}]}`},
		{connection.OpenRouter, `{"data":[{"id":"m","reasoning":{"default_enabled":false}}]}`},
	} {
		t.Run(string(test.kind), func(t *testing.T) {
			c := mockClient(t, test.kind, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/ps" {
					fmt.Fprint(w, `{"models":[]}`)
					return
				}
				fmt.Fprint(w, test.body)
			})
			info, err := prepareModel(c, context.Background(), "m")
			if err != nil || info.Reasoning == nil || info.Reasoning.DefaultOn == nil || *info.Reasoning.DefaultOn {
				t.Fatalf("metadata=%+v err=%v", info, err)
			}
		})
	}
}
