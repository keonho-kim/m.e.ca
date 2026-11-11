// Purpose: Preserve masked narrative-only views and reject malformed tool results without silent fallback.
// Pattern: Public detail projection boundary tests.
// Usage: go test ./internal/serverapi/detail -run TestView.
// Related: internal/serverapi/detail/view.go, internal/serverapi/detail/write_string.go.
package detail

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestViewNarrativesContainOnlyTheSelectedMaskedText(t *testing.T) {
	vault := &credentials.Vault{}
	vault.Remember("private-key")
	for _, test := range []struct {
		name     string
		view     server.DetailView
		original server.DetailOriginal
	}{
		{"message", server.DetailText, server.DetailOriginal{History: &history.Record{Kind: "message", Message: model.Message{Content: "한글 private-key", Reasoning: "other-field"}}}},
		{"event-text", server.DetailText, server.DetailOriginal{History: &history.Record{Kind: "event", Event: record.Event{Text: "한글 private-key", Thinking: "other-field"}}}},
		{"thinking", server.DetailThinking, server.DetailOriginal{History: &history.Record{Kind: "event", Event: record.Event{Text: "other-field", Thinking: "한글 private-key"}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := writeView(t.Context(), &output, test.original, test.view, vault.NewChunkMask); err != nil {
				t.Fatal(err)
			}
			if actual := output.String(); actual != "한글 [redacted]" || strings.Contains(actual, "other-field") {
				t.Fatal("narrative projection mixed fields or skipped masking", actual)
			}
		})
	}
}

func TestViewRejectsInvalidResultAndMissingMask(t *testing.T) {
	vault := &credentials.Vault{}
	original := server.DetailOriginal{Tool: &record.Event{Text: "fallback must not hide corruption", Output: json.RawMessage(`{"broken"`)}}
	var output bytes.Buffer
	if err := writeView(t.Context(), &output, original, server.DetailToolResult, vault.NewChunkMask); err == nil {
		t.Fatal("invalid result was silently replaced with a display summary")
	}
	if err := writeView(t.Context(), &output, original, server.DetailToolResult, nil); err == nil {
		t.Fatal("detail accepted a missing redaction boundary")
	}
}
