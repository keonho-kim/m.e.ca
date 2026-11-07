// Purpose: Lock down public detail fields and mask structured output without leaking private runtime state.
// Pattern: Public wire document contract tests.
// Usage: go test ./internal/serverapi/detail -run TestDocument.
// Related: internal/serverapi/detail/document.go, internal/serverapi/detail/event.go.
package detail

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestDocumentExcludesPrivateMessageAndEventFields(t *testing.T) {
	vault := &credentials.Vault{}
	for _, original := range []server.DetailOriginal{
		{History: &history.Record{Sequence: 7, Kind: "message", Message: model.Message{Role: "assistant", Content: "visible", Reasoning: "private-reasoning", ReasoningState: []byte("private-state"), ProviderID: "private-provider", Calls: []tool.Call{{ID: "private-call"}}}}},
		{Tool: &record.Event{Kind: "tool_succeeded", Text: "visible", Capture: json.RawMessage(`{"private-capture":true}`), Checkpoint: &record.Checkpoint{}, Operation: &tool.Operation{Name: "execute_command", AffectedPaths: []string{"private-target"}}}},
	} {
		body, err := marshalDocument(original, vault.Redact)
		if err != nil {
			t.Fatal(err)
		}
		if !json.Valid(body) || !bytes.Contains(body, []byte(`"kind"`)) || !bytes.Contains(body, []byte("visible")) {
			t.Fatal(string(body))
		}
		for _, forbidden := range []string{"private-", "reasoning_state", "checkpoint", "capture", "AffectedPaths", "Message", "Event"} {
			if bytes.Contains(body, []byte(forbidden)) {
				t.Fatalf("private or internal field %q escaped the allowlist", forbidden)
			}
		}
	}
}

func TestDocumentMasksEscapedSecretsAndPreservesNumbers(t *testing.T) {
	vault := &credentials.Vault{}
	vault.Remember("private-key")
	body, err := marshalDocument(server.DetailOriginal{Tool: &record.Event{Kind: "tool_succeeded", Output: json.RawMessage(`{"key":"private-\u006bey","n":9007199254740993}`)}}, vault.Redact)
	if err != nil || !bytes.Contains(body, []byte("[redacted]")) || !bytes.Contains(body, []byte("9007199254740993")) {
		t.Fatal(string(body), err)
	}
}
