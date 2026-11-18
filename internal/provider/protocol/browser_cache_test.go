// Purpose: Reuse stable request encodings without retaining fresh browser pages or screenshots.
// Pattern: Wire parity and allocation verification with unchanged conversation history.
// Usage: go test ./internal/provider/protocol -run TestBrowserProviderPrefix -bench BenchmarkBrowserProviderPrefix -benchmem.
// Related: internal/provider/protocol/source_history.go, internal/provider/protocol/cloud_sources.go.
package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestBrowserProviderPrefix(t *testing.T) {
	image, err := media.New("shot.png", "image/png", "png", 8, 8, []byte("pixels"))
	if err != nil {
		t.Fatal(err)
	}
	inspect := toolapi.Call{ID: "observe", Name: toolapi.BrowserInspect, Arguments: json.RawMessage(`{"view":"screenshot"}`)}
	skill := toolapi.Call{ID: "skill", Name: "list_skills", Arguments: json.RawMessage(`{}`)}
	for _, kind := range []connection.ProviderKind{connection.LiteLLM, connection.OpenAI, connection.Claude, connection.Gemini} {
		t.Run(string(kind), func(t *testing.T) {
			var encoder Encoder
			stable := []modelapi.Message{
				{Role: "system", Content: "Rules"},
				{Role: "user", Content: strings.Repeat("earlier findings\n", 8192)},
				{Role: "assistant", Calls: []toolapi.Call{inspect, skill}},
			}
			source := append(stable,
				modelapi.Message{Role: "tool", ToolName: inspect.Name, ToolCallID: inspect.ID, Content: `{"text":"viewport"}`, Images: []media.Image{image}, TransientObservation: true},
				modelapi.Message{Role: "tool", ToolName: skill.Name, ToolCallID: skill.ID, Content: `{}`},
				modelapi.Message{Role: "user", Content: "Continue."},
			)
			checkPreparedSource(t, &encoder, kind, source)
			var wire []byte
			if kind == connection.Claude || kind == connection.Gemini {
				if len(encoder.cloudGroups) != len(stable) || len(encoder.cloud) != len(stable) {
					t.Fatal("multimodal observation displaced the stable cache or entered it")
				}
				wire = encoder.cloud[1].parts
			} else {
				if len(encoder.groups) != len(stable) || len(encoder.history) != len(stable) {
					t.Fatal("native tool-result group displaced the stable cache or entered it")
				}
				wire = encoder.history[1].json
			}
			source[3].Images = nil
			source[3].Content = `{"text":"next fresh page"}`
			checkPreparedSource(t, &encoder, kind, source)
			var next []byte
			if kind == connection.Claude || kind == connection.Gemini {
				next = encoder.cloud[1].parts
			} else {
				next = encoder.history[1].json
			}
			if len(wire) == 0 || &wire[0] != &next[0] {
				t.Fatal("stable prefix re-encoded")
			}
		})
	}
}

func BenchmarkBrowserProviderPrefix(b *testing.B) {
	source := []modelapi.Message{{Role: "system", Content: "Rules"}}
	for i := range 128 {
		role := "user"
		if i%2 > 0 {
			role = "assistant"
		}
		source = append(source, modelapi.Message{Role: role, Content: strings.Repeat("observed input.\n", 256)})
	}
	call := toolapi.Call{ID: "read", Name: toolapi.BrowserInspect, Arguments: json.RawMessage(`{"view":"text"}`)}
	source = append(source, modelapi.Message{Role: "assistant", Calls: []toolapi.Call{call}},
		modelapi.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: strings.Repeat("fresh page\n", 512), TransientObservation: true})
	request := modelapi.Request{ModelID: "model", MaxOutput: 8192}
	for _, kind := range []connection.ProviderKind{connection.LiteLLM, connection.Claude} {
		for _, reuse := range []bool{false, true} {
			mode := "uncached"
			if reuse {
				mode = "warm-prefix"
			}
			b.Run(string(kind)+"/"+mode, func(b *testing.B) {
				var encoder *Encoder
				if reuse {
					encoder = new(Encoder)
				}
				encode := func() int {
					messages, err := encoder.Messages(kind, request.ModelID, source)
					if err != nil {
						b.Fatal(err)
					}
					_, body, err := encoder.Encode(kind, "https://fixture.local", nil, request, messages, Controls{})
					if err != nil {
						b.Fatal(err)
					}
					return body.Len()
				}
				b.SetBytes(int64(encode()))
				b.ReportAllocs()
				for b.Loop() {
					encode()
				}
			})
		}
	}
}
