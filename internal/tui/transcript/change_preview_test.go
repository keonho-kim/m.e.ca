// Purpose: Verify applied file contents remain visible with aligned line numbers and safe status handling.
// Pattern: Transcript result contract tests.
// Usage: go test ./internal/tui/transcript -run TestAppliedFile.
// Related: internal/tui/transcript/file_results.go, internal/tui/transcript/change_preview.go.
package transcript

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"strings"
	"testing"
)

func TestAppliedFileContentsAreVisible(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return strings.ReplaceAll(s, "secret", "[redacted]") }}
	for _, name := range []string{"write_file", "edit_file"} {
		event := runrecord.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: name, Arguments: []byte(`{"path":"a"}`)}, Output: []byte(`{"changed":true}`), ChangePreview: "--- a\n+++ a\n@@ -1,1 +1,1 @@\n-old\n+secret\n@@ -100,1 +100,1 @@\n-last\n+new\n"}
		shown := f.Tool(event)
		for _, want := range []string{"-   1 | old", "+   1 | [redacted]", "- 100 | last", "+ 100 | new"} {
			if !strings.Contains(shown, want) {
				t.Fatal(shown)
			}
		}
		restored := f.Event(event)
		if len(restored) != 1 || restored[0].Text != shown {
			t.Fatal("restored content differs")
		}
		event.Kind = "tool_failed"
		event.Text = "write failed"
		shown = f.Tool(event)
		if strings.Contains(shown, "+   1 |") || !strings.Contains(shown, "write failed") {
			t.Fatal("failed change presented as applied")
		}
	}
}

func TestAppliedFilePreservesNoNewlineAndLiteralHeaders(t *testing.T) {
	preview := "--- /dev/null\n+++ a\n@@ -0,0 +1,2 @@\n+# Heading\n+++literal\n\\ No newline at end of file\n"
	shown := renderChangePreview(preview, locale.Korean).text
	if !strings.Contains(shown, "+   1 | # Heading") || !strings.Contains(shown, "+   2 | ++literal") || !strings.Contains(shown, "마지막 줄바꿈 없음") {
		t.Fatal(shown)
	}
}
