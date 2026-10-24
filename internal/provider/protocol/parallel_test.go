// Purpose: Verify supported request dialects advertise native multi-call support.
// Pattern: Wire contract tests.
// Usage: go test ./internal/provider/protocol -run TestParallel.
// Related: internal/provider/protocol/request.go, internal/provider/protocol/lmstudio.go.
package protocol

import (
	"encoding/json"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"testing"
)

func TestParallelNativeRequests(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.VLLM, connection.LMStudio, connection.Ollama} {
		t.Run(string(kind), func(t *testing.T) {
			_, body, err := Encode(kind, "http://localhost", nil, modelapi.Request{ModelID: "m", MaxOutput: 100, Tools: []toolapi.Definition{{Name: "inspect_fixture", Description: "Read", Parameters: []toolapi.Parameter{{Name: "path", Kind: toolapi.Text, Required: true, MaxBytes: 4096}}}}}, []Message{{Role: "user", Content: "read files"}}, Controls{})
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				Parallel *bool `json:"parallel_tool_calls"`
			}
			if err := json.Unmarshal(body, &wire); err != nil {
				t.Fatal(err)
			}
			if kind == connection.Ollama {
				if wire.Parallel != nil {
					t.Fatal("unsupported Ollama field")
				}
			} else if wire.Parallel == nil || !*wire.Parallel {
				t.Fatalf("parallel calls disabled: %s", body)
			}
		})
	}
}
