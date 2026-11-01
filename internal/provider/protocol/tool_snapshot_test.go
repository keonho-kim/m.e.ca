// Purpose: Verify tool schema reuse preserves every dialect, mutable input and replay boundary.
// Pattern: Differential protocol and ownership regression tests.
// Usage: Run TestToolSnapshot with the protocol test suite.
// Related: internal/provider/protocol/tool_snapshot.go, internal/provider/protocol/header.go.
package protocol

import (
	"bytes"
	"io"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestToolSnapshotReuseMutationAndReplay(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.VLLM, connection.SGLang, connection.LiteLLM, connection.Ollama, connection.LMStudio} {
		t.Run(string(kind), func(t *testing.T) {
			var encoder Encoder
			defer encoder.Clear()
			request := modelapi.Request{ModelID: "model", MaxOutput: 100, Tools: []toolapi.Definition{{Name: "inspect", Description: "initial", Parameters: []toolapi.Parameter{{Name: "entries", Kind: toolapi.ObjectList, Max: 4, Fields: []toolapi.Parameter{{Name: "format", Kind: toolapi.Text, Values: []string{"first"}, MaxBytes: 20}}}}}}}
			source := []modelapi.Message{{Role: "user", Content: "small request"}}
			messages, err := encoder.Messages(kind, request.ModelID, source)
			if err != nil {
				t.Fatal(err)
			}
			_, oldBody, err := encoder.Encode(kind, "https://model.test", nil, request, messages, Controls{})
			if err != nil {
				t.Fatal(err)
			}
			original, err := io.ReadAll(oldBody.Reader())
			if err != nil {
				t.Fatal(err)
			}
			originalTools := encoder.tools.encoded
			if len(originalTools) == 0 {
				t.Fatal("small request did not retain tool schema")
			}
			messages, err = encoder.Messages(kind, request.ModelID, source)
			if err != nil {
				t.Fatal(err)
			}
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			if &encoder.tools.encoded[0] != &originalTools[0] {
				t.Fatal("unchanged schemas were rebuilt")
			}
			for _, change := range []func(){
				func() { request.Tools[0].Description = "changed description" },
				func() { request.Tools[0].Parameters[0].Fields[0].Values[0] = "second" },
				func() { request.Tools[0].Parameters[0].Fields[0].Required = true },
			} {
				previous := encoder.tools.encoded
				change()
				assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
				if bytes.Equal(previous, encoder.tools.encoded) {
					t.Fatal("schema mutation reused stale wire JSON")
				}
			}
			temperature := 0.7
			request.MaxOutput, request.Temperature = 57, &temperature
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			request.Tools = nil
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			if encoder.tools.encoded != nil || encoder.tools.definitions != nil {
				t.Fatal("removed tools remained retained")
			}
			encoder.Clear()
			replayed, err := io.ReadAll(oldBody.Reader())
			if err != nil || !bytes.Equal(original, replayed) {
				t.Fatal("later settings changed an earlier replayable body", err)
			}
		})
	}
}

func TestToolSnapshotBoundIsNotAnAcceptanceLimit(t *testing.T) {
	var encoder Encoder
	request := modelapi.Request{ModelID: "model", MaxOutput: 100, Tools: []toolapi.Definition{{Name: "inspect", Description: strings.Repeat("description ", toolCacheCapacity/8)}}}
	defer encoder.Clear()
	messages := []Message{{Role: "user", Content: "request"}}
	assertEncodedRequest(t, &encoder, connection.OpenRouter, request, messages, Controls{})
	if encoder.tools.bytes != 0 || encoder.tools.encoded != nil || encoder.tools.definitions != nil {
		t.Fatal("oversized accepted schema entered the bounded cache")
	}
}
