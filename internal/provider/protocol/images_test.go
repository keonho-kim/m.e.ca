// Purpose: Verify image payload dialects and preserve native multi-tool result ordering.
// Pattern: Provider request contract tests.
// Usage: go test ./internal/provider/protocol -run TestImage.
// Related: internal/provider/protocol/input.go, internal/provider/protocol/images.go.
package protocol

import (
	"encoding/base64"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"strings"
	"testing"
)

func TestImageRequestDialectsAndNativeOrdering(t *testing.T) {
	picture, err := media.New("shot.png", "image/png", "png", 16, 16, []byte("immutable-image-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	calls := []toolapi.Call{{ID: "image-call", Name: "read_image", Arguments: json.RawMessage(`{"path":"shot.png"}`)}, {ID: "other-call", Name: "execute_command", Arguments: json.RawMessage(`{"command":"pwd"}`)}}
	source := []modelapi.Message{{Role: "assistant", Calls: calls}, {Role: "tool", ToolName: "read_image", ToolCallID: "image-call", Content: "dimensions", Images: []media.Image{picture}}, {Role: "tool", ToolName: "execute_command", ToolCallID: "other-call", Content: "/project"}, {Role: "assistant", Content: "Seen"}}
	for _, kind := range []connection.ProviderKind{connection.VLLM, connection.SGLang, connection.LiteLLM, connection.OpenRouter, connection.Ollama, connection.LMStudio} {
		t.Run(string(kind), func(t *testing.T) {
			messages, err := Messages(kind, source)
			if err != nil || len(messages) != 5 || messages[1].Role != "tool" || messages[2].Role != "tool" || messages[3].Role != "user" {
				t.Fatalf("ordering: %+v %v", messages, err)
			}
			_, body, err := Encode(kind, "http://model", nil, modelapi.Request{ModelID: "m", MaxOutput: 32}, messages, Controls{})
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]json.RawMessage
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			encoded := base64.StdEncoding.EncodeToString(picture.Data)
			if !strings.Contains(string(body), encoded) {
				t.Fatal("image bytes missing")
			}
			switch kind {
			case connection.Ollama:
				if len(messages[3].Images) != 1 || strings.Contains(string(body), "data:image/") {
					t.Fatal("Ollama needs bare base64 images")
				}
			case connection.LMStudio:
				if !strings.Contains(string(body), `"type":"input_image"`) || !strings.Contains(string(body), `"type":"function_call_output"`) {
					t.Fatal("Responses image or tool result missing")
				}
			default:
				if !strings.Contains(string(body), `"type":"image_url"`) {
					t.Fatal("chat image part missing")
				}
			}
		})
	}
}
