// Purpose: Preserve multimodal native groups while reusing immutable image encodings.
// Pattern: Differential protocol and bounded-resource tests.
// Usage: go test ./internal/provider/protocol -run TestImageHistory -bench BenchmarkImageHistory.
// Related: internal/provider/protocol/source_history.go, internal/provider/protocol/images.go.
package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestImageHistoryReusesWholeNativeGroups(t *testing.T) {
	picture, err := media.New("image.png", "image/png", "png", 1, 1, []byte(strings.Repeat("x", 128*1024)))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.VLLM, connection.SGLang, connection.LiteLLM, connection.LMStudio, connection.Ollama} {
		t.Run(string(kind), func(t *testing.T) {
			var encoder Encoder
			t.Cleanup(encoder.Clear)
			source := []modelapi.Message{{Role: "assistant", Calls: []toolapi.Call{{ID: "a", Name: "read_image", Arguments: []byte(`{"path":"image.png"}`)}}}, {Role: "tool", ToolCallID: "a", ToolName: "read_image", Content: "image", Images: []media.Image{picture}}, {Role: "assistant", Content: "observed"}}
			request := modelapi.Request{ModelID: "model", MaxOutput: 32}
			var previous []byte
			for step := range 5 {
				if step == 2 {
					source[1].Images[0].Path = "renamed.png"
				}
				if step == 3 {
					source[1].Images = nil
				}
				plain, err := Messages(kind, source)
				if err != nil {
					t.Fatal(err)
				}
				_, want, err := Encode(kind, "https://model", nil, request, plain, Controls{})
				if err != nil {
					t.Fatal(err)
				}
				cached, err := encoder.Messages(kind, "model", source)
				if err != nil {
					t.Fatal(err)
				}
				if len(cached) != len(plain) {
					t.Fatal("image group changed message count")
				}
				for i := range cached {
					if cached[i].HasImages() != plain[i].HasImages() {
						t.Fatal("cached images would enter text tokenization")
					}
				}
				_, body, err := encoder.Encode(kind, "https://model", nil, request, cached, Controls{})
				if err != nil {
					t.Fatal(err)
				}
				got, err := io.ReadAll(body.Reader())
				var a, b any
				if err != nil || json.Unmarshal(want, &a) != nil || json.Unmarshal(got, &b) != nil || !reflect.DeepEqual(a, b) {
					t.Fatal("cached multimodal request differs", err)
				}
				if step == 1 && &previous[0] != &encoder.history[2].json[0] {
					t.Fatal("image was encoded again")
				}
				if step < 2 {
					previous = encoder.history[2].json
				}
				if step == 4 {
					encoder.Clear()
					if encoder.groups != nil || encoder.history != nil {
						t.Fatal("image cache survived clear")
					}
				}
			}
		})
	}
}

func BenchmarkImageHistory(b *testing.B) {
	for _, cached := range []bool{false, true} {
		b.Run(fmt.Sprint("cached=", cached), func(b *testing.B) {
			picture, err := media.New("image.png", "image/png", "png", 1, 1, make([]byte, 1<<20))
			if err != nil {
				b.Fatal(err)
			}
			source := []modelapi.Message{{Role: "user", Content: "inspect", Images: []media.Image{picture}}}
			request := modelapi.Request{ModelID: "model", MaxOutput: 32}
			var encoder Encoder
			defer encoder.Clear()
			encode := func() {
				var p *Encoder
				if cached {
					p = &encoder
				}
				messages, err := p.Messages(connection.OpenRouter, "model", source)
				if err != nil {
					b.Fatal(err)
				}
				_, _, err = p.Encode(connection.OpenRouter, "https://model", nil, request, messages, Controls{})
				if err != nil {
					b.Fatal(err)
				}
			}
			encode()
			encode()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				encode()
			}
		})
	}
}
