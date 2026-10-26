// Purpose: Verify localized tool chrome preserves literal payloads and immutable runtime records.
// Pattern: Bilingual projection tests.
// Usage: go test ./internal/tui/transcript -run TestLanguage.
// Related: internal/tui/transcript/messages.go, internal/tui/transcript/process_result.go.
package transcript

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestLanguagePreservesLiteralOutput(t *testing.T) {
	for _, lang := range []locale.Language{locale.Korean, locale.English} {
		content := json.RawMessage(`{"Stdout":"allow-all Approve OUTPUT 추론","ExitCode":0,"Status":"exited"}`)
		original := bytes.Clone(content)
		event := runrecord.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "execute_command", Arguments: json.RawMessage(`{"command":"printf literal"}`)}, Output: content}
		f := Formatter{Language: lang, Redact: func(text string) string { return text }}
		shown := f.Tool(event)
		if !strings.Contains(shown, lang.Text("Output")) || !strings.Contains(shown, "allow-all Approve OUTPUT 추론") {
			t.Fatal(shown)
		}
		if !bytes.Equal(content, original) || event.Call.Name != "execute_command" {
			t.Fatal("runtime record translated")
		}
		if user := f.User("allow-all Approve OUTPUT 추론"); !strings.HasPrefix(user, "> allow-all Approve OUTPUT 추론\n\n") {
			t.Fatal(user)
		}
	}
}
