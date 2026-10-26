// Purpose: Verify line-level approval previews preserve actual changes and omit distant unchanged content.
// Pattern: Behavioral tests.
// Usage: go test ./internal/tools/local/diff.
// Related: internal/tools/local/diff/diff.go, internal/tools/local/executor.go.
package diff

import (
	"fmt"
	"strings"
	"testing"
)

func TestPreview(t *testing.T) {
	for _, test := range []struct {
		name, before, after string
		old, next           bool
		want                []string
	}{
		{"edit", "same\nold\nend\n", "same\nnew\nend\n", true, true, []string{"--- file\n+++ file\n", "@@ -1,3 +1,3 @@", "-old\n+new\n", " same\n", " end\n"}},
		{"create", "", "new\n", false, true, []string{"--- /dev/null\n+++ file\n", "@@ -0,0 +1,1 @@", "+new\n"}},
		{"delete", "old\n", "", true, false, []string{"--- file\n+++ /dev/null\n", "@@ -1,1 +0,0 @@", "-old\n"}},
		{"newline", "same", "same\n", true, true, []string{"-same\n\\ No newline at end of file\n+same\n"}},
		{"CRLF", "same\r\n", "same\n", true, true, []string{"-same\\r\n+same\n"}},
		{"unchanged", "same\n", "same\n", true, true, []string{"No content changes."}},
		{"binary", "\x00old", "new", true, true, []string{"Binary content changed:", "4 bytes", "3 bytes"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			preview := Preview("file", []byte(test.before), []byte(test.after), test.old, test.next)
			for _, want := range test.want {
				if !strings.Contains(preview, want) {
					t.Fatalf("missing %q in %q", want, preview)
				}
			}
		})
	}
}
func TestLargePreviewDoesNotDropChanges(t *testing.T) {
	preview := Preview("file", []byte(strings.Repeat("old\n", 65537)), []byte("new\n"), true, true)
	if strings.Count(preview, "\n-old") != 65537 || !strings.Contains(preview, "@@ -1,65537 +1,1 @@") || !strings.Contains(preview, "+new\n") {
		t.Fatal("large-file fallback omitted changed lines")
	}
}
func TestPreviewSeparateHunks(t *testing.T) {
	var lines []string
	for i := 1; i <= 25; i++ {
		lines = append(lines, fmt.Sprintf("line %d\n", i))
	}
	before := strings.Join(lines, "")
	lines[2], lines[21] = "changed 3\n", "changed 22\n"
	preview := Preview("file", []byte(before), []byte(strings.Join(lines, "")), true, true)
	if strings.Count(preview, "\n@@ ") != 2 || strings.Contains(preview, "line 12") {
		t.Fatalf("unchanged middle was not omitted: %s", preview)
	}
	for _, change := range []string{"-line 3\n+changed 3\n", "-line 22\n+changed 22\n"} {
		if !strings.Contains(preview, change) {
			t.Fatal("lost changed lines")
		}
	}
}
func TestMatchingPreservesBothVersions(t *testing.T) {
	for _, pair := range [][2]string{
		{"a\nb\na\nc\n", "b\na\na\nd\n"},
		{"", "new\n"}, {"old\n", ""},
		{"a\nb\n", "x\na\nb\ny"},
		{strings.Repeat("old\n", 300), strings.Repeat("new\n", 300)},
	} {
		var old, next strings.Builder
		ops, _ := match(split(pair[0]), split(pair[1]))
		for _, op := range ops {
			if op.kind != '+' {
				old.WriteString(op.text)
			}
			if op.kind != '-' {
				next.WriteString(op.text)
			}
		}
		if old.String() != pair[0] || next.String() != pair[1] {
			t.Fatal("diff did not preserve both complete file versions")
		}
	}
}
