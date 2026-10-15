// Purpose: Verify user tool displays show line numbers while runtime records retain exact anchors.
// Pattern: Transcript boundary regression tests.
// Usage: go test ./internal/tui/transcript -run TestAnchorDisplay.
// Related: internal/tui/transcript/anchors.go, internal/tui/transcript/tools.go.
package transcript

import (
	"bytes"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestAnchorDisplayReadKeepsSourceAndModelRecord(t *testing.T) {
	raw := []byte(`{"Content":"7#1234abcd | value = 3\n8#8765abcd | #deadbeef | actual source\n9 | partial","Note":"Copy line#8hex anchors","StartLine":7}`)
	original := bytes.Clone(raw)
	event := run.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "read_file", Arguments: []byte(`{"path":"a"}`)}, Output: raw}
	f := Formatter{Redact: func(s string) string { return s }}
	shown := f.Tool(event)
	for _, want := range []string{"7 | value = 3", "8 | #deadbeef | actual source", "9 | partial"} {
		if !strings.Contains(shown, want) {
			t.Fatalf("missing %q: %s", want, shown)
		}
	}
	if strings.Contains(shown, "1234abcd") || strings.Contains(shown, "8765abcd") || strings.Contains(shown, "line#8hex") {
		t.Fatal(shown)
	}
	if !bytes.Equal(raw, original) {
		t.Fatal("display changed the model's original output")
	}
	blocks := f.Event(event)
	if len(blocks) != 1 || blocks[0].Text != shown {
		t.Fatal("restored display differs")
	}
}
func TestAnchorDisplayEditArgumentsKeepRuntimeBinding(t *testing.T) {
	raw := []byte(`{"path":"a","start_anchor":"7#1234abcd","end_anchor":"9#8765abcd","new_text":"replacement"}`)
	original := bytes.Clone(raw)
	shown := (Formatter{Redact: func(s string) string { return s }}).Tool(run.Event{Kind: "tool_started", Call: &tool.Call{Name: "edit_file", Arguments: raw}})
	if !strings.Contains(shown, "start_line: 7") || !strings.Contains(shown, "end_line: 9") || strings.Contains(shown, "1234abcd") {
		t.Fatal(shown)
	}
	if !bytes.Equal(raw, original) {
		t.Fatal("display changed executable arguments")
	}
}
