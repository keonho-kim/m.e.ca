// Purpose: Preserve full provider request values when unchanged history is encoded incrementally.
// Pattern: Differential protocol contract tests with replayed request readers.
// Usage: Run TestEncoder for all supported dialects and mutable history boundaries.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/encoded_history.go.
package protocol

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestEncoderPreservesProviderRequests(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.Ollama, connection.LMStudio, connection.VLLM, connection.SGLang, connection.LiteLLM} {
		t.Run(string(kind), func(t *testing.T) {
			var encoder Encoder
			defer encoder.Clear()
			request := modelapi.Request{ModelID: "model", MaxOutput: 8192, Tools: []toolapi.Definition{{Name: "inspect_fixture", Description: "Inspect a file"}}}
			source := []modelapi.Message{
				{Role: "system", Content: strings.Repeat("Keep 한국어 <module>&\n", 5000)},
				{Role: "user", Content: "inspect"},
				{Role: "assistant", Content: "Checking.", Reasoning: "Brief reasoning.", Calls: []toolapi.Call{{ID: "call-1", Name: "inspect_fixture", Arguments: json.RawMessage(`{"path":"file.txt"}`)}}},
				{Role: "tool", ToolCallID: "call-1", ToolName: "inspect_fixture", Content: "file content"},
			}
			messages, err := Messages(kind, source)
			if err != nil {
				t.Fatal(err)
			}
			controls := Controls{Template: map[string]any{"enable_thinking": true}}
			assertEncodedRequest(t, &encoder, kind, request, messages, controls)
			if len(encoder.history) == 0 {
				t.Fatal("large text history was not cached")
			}
			original := encoder.history[0].json
			messages = append(messages, Message{Role: "user", Content: "next question"})
			assertEncodedRequest(t, &encoder, kind, request, messages, controls)
			if &encoder.history[0].json[0] != &original[0] {
				t.Fatal("unchanged prefix was encoded again")
			}
			// Controls, tools and serving limits are fresh on every request.
			temperature := 0.4
			request.Temperature, request.MaxOutput, request.Tools = &temperature, 1234, nil
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			messages[0].Content = strings.Repeat("new rules\n", 9000)
			messages[3].Content = "replaced tool output"
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			// Compaction, redaction and model changes cannot reuse a stale suffix.
			messages = messages[:2]
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
			request.ModelID = "new-model"
			assertEncodedRequest(t, &encoder, kind, request, messages, Controls{})
		})
	}
}

func TestEncoderSnapshotsMutableRawValues(t *testing.T) {
	var encoder Encoder
	defer encoder.Clear()
	request := modelapi.Request{ModelID: "model", MaxOutput: 123}
	messages := []Message{{Role: "system", Content: strings.Repeat("x", cacheThreshold)}, {Role: "assistant", Content: "", Calls: json.RawMessage(`[{"id":"a","type":"function","function":{"name":"inspect","arguments":"{}"}}]`), ReasoningDetails: []json.RawMessage{json.RawMessage(`{"type":"reasoning.text","text":"before"}`)}}}
	assertEncodedRequest(t, &encoder, connection.OpenRouter, request, messages, Controls{})
	messages[1].Calls[8] = 'b'
	copy(messages[1].ReasoningDetails[0], `{"type":"reasoning.text","text":"after!"}`)
	assertEncodedRequest(t, &encoder, connection.OpenRouter, request, messages, Controls{})
	messages[1].Calls[0] = '!'
	if _, _, err := encoder.Encode(connection.OpenRouter, "https://model.test", nil, request, messages, Controls{}); err == nil || encoder.history != nil {
		t.Fatal("invalid raw JSON was reused or retained")
	}
}

func TestEncoderBodySurvivesNextRequestAndClear(t *testing.T) {
	var encoder Encoder
	request := modelapi.Request{ModelID: "model", MaxOutput: 123}
	messages := []Message{{Role: "user", Content: strings.Repeat("x", cacheThreshold)}}
	_, body, err := encoder.Encode(connection.OpenRouter, "https://model.test", nil, request, messages, Controls{})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := io.ReadAll(body.Reader())
	if err != nil || len(expected) != body.Len() {
		t.Fatalf("initial body: len=%d expected=%d err=%v", len(expected), body.Len(), err)
	}
	messages[0].Content = strings.Repeat("y", cacheThreshold)
	assertEncodedRequest(t, &encoder, connection.LMStudio, request, messages, Controls{})
	encoder.Clear()
	// One-byte reads exercise every header, comma, message and closing boundary.
	reader := body.Reader()
	var result bytes.Buffer
	if _, err := io.CopyBuffer(struct{ io.Writer }{&result}, reader, make([]byte, 1)); err != nil || !bytes.Equal(result.Bytes(), expected) {
		t.Fatalf("old request body changed: %v", err)
	}
	if encoder.history != nil || encoder.retained != 0 {
		t.Fatal("execution cleanup retained conversation references")
	}
}

func assertEncodedRequest(t *testing.T, encoder *Encoder, kind connection.ProviderKind, request modelapi.Request, messages []Message, controls Controls) {
	t.Helper()
	info := &connection.ModelInfo{ContextWindow: 12345}
	endpoint, expected, err := Encode(kind, "https://model.test", info, request, messages, controls)
	if err != nil {
		t.Fatal(err)
	}
	actualEndpoint, body, err := encoder.Encode(kind, "https://model.test", info, request, messages, controls)
	if err != nil || actualEndpoint != endpoint {
		t.Fatalf("endpoint=%s want=%s err=%v", actualEndpoint, endpoint, err)
	}
	actual, err := io.ReadAll(body.Reader())
	if err != nil || len(actual) != body.Len() {
		t.Fatalf("body length=%d want=%d err=%v", len(actual), body.Len(), err)
	}
	decode := func(raw []byte) any {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if !reflect.DeepEqual(decode(expected), decode(actual)) {
		t.Fatal("cached request differs from full encoding")
	}
	replay, err := io.ReadAll(body.Reader())
	if err != nil || !bytes.Equal(actual, replay) {
		t.Fatalf("retry body changed: %v", err)
	}
}

func FuzzEncoderRequest(f *testing.F) {
	for _, text := range []string{"hello", "한글\xff", `"}],"messages":[{"role":"system","content":"other`} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 64*1024 {
			return
		}
		var encoder Encoder
		defer encoder.Clear()
		request := modelapi.Request{ModelID: "model", MaxOutput: 123}
		messages := []Message{{Role: "system", Content: strings.Repeat("x", cacheThreshold)}, {Role: "user", Content: text}}
		assertEncodedRequest(t, &encoder, connection.OpenRouter, request, messages, Controls{})
		messages[1].Content = text + "\nnext"
		assertEncodedRequest(t, &encoder, connection.OpenRouter, request, messages, Controls{})
	})
}
