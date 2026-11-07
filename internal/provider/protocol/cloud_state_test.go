// Purpose: Preserve empty signed blocks and reject changed Gemini tool arguments during replay.
// Pattern: Native continuation regression tests.
// Usage: go test ./internal/provider/protocol -run TestCloudContinuation.
// Related: internal/provider/protocol/claude/types.go, internal/provider/protocol/gemini_request.go.
package protocol

import (
	"encoding/json"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol/claude"
	"github.com/keonho-kim/meca/internal/provider/protocol/gemini"
)

func TestCloudContinuationKeepsEmptySignedFields(t *testing.T) {
	block := claude.Block{Type: "thinking", Signature: "signature"}
	body, err := json.Marshal(block)
	if err != nil || !strings.Contains(string(body), `"thinking":""`) {
		t.Fatal("omitted Claude thinking field", string(body), err)
	}
	var part gemini.Part
	if err := json.Unmarshal([]byte(`{"text":"","thoughtSignature":"signature"}`), &part); err != nil {
		t.Fatal(err)
	}
	body, err = json.Marshal(part)
	if err != nil || !strings.Contains(string(body), `"text":""`) {
		t.Fatal("omitted Gemini signed empty text", string(body), err)
	}
}

func TestCloudContinuationRejectsChangedSignedCall(t *testing.T) {
	message := Message{Role: "assistant", Content: "", Calls: json.RawMessage(`[{"id":"local","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"changed\"}"}}]`), continuation: json.RawMessage(`{"parts":[{"functionCall":{"name":"read_file","args":{"path":"original"}},"thoughtSignature":"signature"}],"call_ids":["local"]}`)}
	if _, _, err := Encode(connection.Gemini, "https://example.com/v1beta", nil, modelapi.Request{ModelID: "alias", MaxOutput: 128}, []Message{message}, Controls{}); err == nil {
		t.Fatal("signed call was replayed with different arguments")
	}
}

func TestCloudGeminiContinuationRejectsDifferentText(t *testing.T) {
	message := Message{Role: "assistant", Content: "another reply", continuation: json.RawMessage(`{"parts":[{"text":"original reply","thoughtSignature":"signature"}],"call_ids":[]}`)}
	request := modelapi.Request{ModelID: "alias", MaxOutput: 128}
	if _, _, err := Encode(connection.Gemini, "https://example.com/v1beta", nil, request, []Message{message}, Controls{}); err == nil {
		t.Fatal("continuation from another message was accepted")
	}
	message.Content = "original reply"
	if _, _, err := Encode(connection.Gemini, "https://example.com/v1beta", nil, request, []Message{message}, Controls{}); err != nil {
		t.Fatal(err)
	}
}
