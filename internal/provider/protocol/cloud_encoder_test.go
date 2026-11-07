// Purpose: Bound warm native request allocation and preserve signed history through cache invalidation.
// Pattern: Differential provider and resource lifetime tests.
// Usage: go test ./internal/provider/protocol -run TestCloudEncoder.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/claude_history_test.go.
package protocol

import (
	"bytes"
	"encoding/json"
	"io"
	"runtime"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestCloudEncoderBoundsWarmRequestAllocation(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.Claude, connection.Gemini} {
		t.Run(string(kind), func(t *testing.T) {
			var encoder Encoder
			defer encoder.Clear()
			request := modelapi.Request{ModelID: "model", MaxOutput: 32}
			messages := []Message{{Role: "system", Content: "rules"}, {Role: "user", Content: strings.Repeat("x", 1<<20)}}
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			_, body, err := encoder.Encode(kind, "https://model.test", nil, request, messages, Controls{})
			runtime.ReadMemStats(&after)
			if err != nil || body.Len() == 0 {
				t.Fatal(err)
			}
			if used := after.TotalAlloc - before.TotalAlloc; used > 1<<20 {
				t.Fatalf("unchanged native history allocated %d bytes", used)
			}
			original, err := io.ReadAll(body.Reader())
			if err != nil {
				t.Fatal(err)
			}
			messages = append(messages, Message{Role: "user", Content: "next"})
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			messages[0].Content = "replacement rules"
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			encoder.Clear()
			replay, err := io.ReadAll(body.Reader())
			if err != nil || !bytes.Equal(original, replay) {
				t.Fatal("retained request changed after invalidation", err)
			}
		})
	}
}

func TestCloudEncoderClaudeInvalidatesSignedPrefix(t *testing.T) {
	for _, change := range []string{"system", "observation", "tools", "model"} {
		t.Run(change, func(t *testing.T) {
			var encoder Encoder
			request := modelapi.Request{ModelID: "alias", MaxOutput: 32}
			messages := []Message{{Role: "system", Content: strings.Repeat("rules ", 12000)}, {Role: "user", Content: "objective"}}
			_, cached, err := encoder.Encode(connection.Claude, "https://model.test", nil, request, messages, Controls{})
			if err != nil {
				t.Fatal(err)
			}
			_, full, err := singleBody(connection.Claude, "https://model.test", nil, request, messages, Controls{})
			probe := []byte(`[{"type":"text","text":"probe"}]`)
			cachedState, cachedErr := cached.BindContinuation(probe)
			fullState, fullErr := full.BindContinuation(probe)
			if err != nil || cachedErr != nil || fullErr != nil || string(cachedState) != string(fullState) {
				t.Fatal("cached prefix differs", err)
			}
			signed, err := cached.BindContinuation([]byte(`[{"type":"text","text":"before"},{"type":"thinking","signature":"opaque","extra":9007199254740993},{"type":"text","text":"after"}]`))
			if err != nil {
				t.Fatal(err)
			}
			messages = append(messages, Message{Role: "assistant", Content: "beforeafter", continuation: signed}, Message{Role: "user", Content: "continue"})
			assertEncodedRequest(t, &encoder, connection.Claude, request, messages, Controls{})
			switch change {
			case "system":
				messages[0].Content = messages[0].Content.(string) + "changed"
			case "observation":
				messages[1].Content = "replaced objective"
			case "tools":
				request.Tools = []toolapi.Definition{{Name: "inspect", Description: "Inspect"}}
			case "model":
				request.ModelID = "another-model"
			}
			assertEncodedRequest(t, &encoder, connection.Claude, request, messages, Controls{})
		})
	}
}

func TestCloudEncoderGeminiRebindsChangedNativeIDs(t *testing.T) {
	var encoder Encoder
	request := modelapi.Request{ModelID: "alias", MaxOutput: 32}
	messages := []Message{
		{Role: "system", Content: strings.Repeat("rules ", 12000)},
		{Role: "assistant", Content: "", nativeCalls: []toolapi.Call{{ID: "local", Name: "inspect", Arguments: json.RawMessage(`{}`)}}, continuation: json.RawMessage(`{"parts":[{"functionCall":{"id":"wire-before","name":"inspect","args":{}},"thoughtSignature":"opaque"}],"call_ids":["local"]}`)},
		{Role: "tool", ToolCallID: "local", ToolName: "inspect", Content: "result"},
	}
	assertEncodedRequest(t, &encoder, connection.Gemini, request, messages, Controls{})
	assertEncodedRequest(t, &encoder, connection.Gemini, request, messages, Controls{})
	messages[1].continuation = bytes.ReplaceAll(messages[1].continuation, []byte("wire-before"), []byte("wire-after"))
	assertEncodedRequest(t, &encoder, connection.Gemini, request, messages, Controls{})
	messages[1].nativeCalls[0].Arguments = json.RawMessage(`{"changed":true}`)
	if _, _, err := encoder.Encode(connection.Gemini, "https://model.test", nil, request, messages, Controls{}); err == nil || encoder.cloud != nil {
		t.Fatal("invalid signed history reused or retained", err)
	}
}

func TestCloudEncoderBudgetAndProviderChangesReleaseHistory(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.Claude, connection.Gemini} {
		encoder := NewEncoder(128 << 10)
		request := modelapi.Request{ModelID: "alias", MaxOutput: 32}
		for _, size := range []int{32768, 65536, 256 << 10, 32} {
			messages := []Message{{Role: "system", Content: strings.Repeat("x", 32768)}, {Role: "user", Content: strings.Repeat("y", size)}}
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			if encoder.cloudRetained > encoder.limit() {
				t.Fatal("native cache exceeded its execution budget")
			}
		}
		if _, err := encoder.Messages(connection.OpenRouter, "alias", []modelapi.Message{{Role: "user", Content: "new provider"}}); err != nil {
			t.Fatal(err)
		}
		if encoder.cloud != nil || encoder.cloudRetained != 0 {
			t.Fatal("another provider retained native history")
		}
		encoder.Clear()
	}
}
