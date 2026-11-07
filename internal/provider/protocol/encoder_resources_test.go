// Purpose: Keep request caching bounded without restricting accepted text or image inputs.
// Pattern: Resource lifetime and fallback contract tests.
// Usage: Run TestEncoderResources and TestEncoderImage after request encoder changes.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/images.go.
package protocol

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestEncoderResources(t *testing.T) {
	var encoder Encoder
	request := modelapi.Request{ModelID: "model", MaxOutput: 123}
	for _, text := range []string{strings.Repeat("x", cacheThreshold), strings.Repeat("\n", cacheCapacity/2+1), strings.Repeat("x", cacheCapacity+1), "small question"} {
		messages := []Message{{Role: "user", Content: text}}
		assertEncodedRequest(t, &encoder, connection.OpenRouter, request, messages, Controls{})
		if len(text) == cacheThreshold {
			if encoder.retained == 0 || encoder.retained > cacheCapacity {
				t.Fatal("expected a bounded text cache")
			}
		} else if encoder.history != nil || encoder.retained != 0 {
			t.Fatal("small or oversized requests retained a cache")
		}
	}
}

func TestEncoderUnpreparedImagesBypassCache(t *testing.T) {
	picture, err := media.New("shot.png", "image/png", "png", 1, 1, []byte("image"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.LMStudio, connection.Ollama} {
		var encoder Encoder
		request := modelapi.Request{ModelID: "model", MaxOutput: 123}
		messages := []Message{{Role: "user", Content: strings.Repeat("x", cacheThreshold)}}
		assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
		messages, err := Messages(kind, []modelapi.Message{{Role: "user", Content: strings.Repeat("x", cacheThreshold), Images: []media.Image{picture}}})
		if err != nil {
			t.Fatal(err)
		}
		assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
		if encoder.history != nil {
			t.Fatal("image input retained text or image encoding cache")
		}
	}
}

func TestEncoderPeerBudgetSurvivesResetWithoutLimitingInput(t *testing.T) {
	const budget = 128 * 1024
	encoder := NewEncoder(budget)
	request := modelapi.Request{ModelID: "peer", MaxOutput: 123}
	for _, size := range []int{70 * 1024, 256 * 1024, 70 * 1024} {
		assertEncodedRequest(t, &encoder, connection.OpenRouter, request, []Message{{Role: "user", Content: strings.Repeat("x", size)}}, Controls{})
		if encoder.retained+encoder.tools.bytes > budget {
			t.Fatal("peer exceeded retained encoding budget")
		}
		encoder.Clear()
		if encoder.limit() != budget {
			t.Fatal("model/reset lost peer cache budget")
		}
	}
}

func TestEncoderProviderSwitchReleasesHistory(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.Claude, connection.Gemini} {
		for _, prepare := range []bool{false, true} {
			var encoder Encoder
			request := modelapi.Request{ModelID: "model", MaxOutput: 32}
			assertEncodedRequest(t, &encoder, connection.OpenRouter, request, []Message{{Role: "user", Content: strings.Repeat("old", cacheThreshold)}}, Controls{})
			messages := []Message{{Role: "user", Content: "new provider"}}
			if prepare {
				if _, err := encoder.Messages(kind, "model", []modelapi.Message{{Role: "user", Content: "new provider"}}); err != nil {
					t.Fatal(err)
				}
			} else if _, _, err := encoder.Encode(kind, "https://example.com", nil, request, messages, Controls{}); err != nil {
				t.Fatal(err)
			}
			if encoder.history != nil || encoder.groups != nil || encoder.retained != 0 || encoder.tools.bytes != 0 {
				t.Fatalf("%s prepare=%v kept another provider's context", kind, prepare)
			}
		}
	}
}
