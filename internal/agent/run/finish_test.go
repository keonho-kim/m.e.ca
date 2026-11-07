// Purpose: Ensure a short final answer does not keep a large discarded whitespace buffer alive.
// Pattern: Final-response resource ownership test.
// Usage: go test ./internal/agent/run -run TestFinalResponseReleasesDiscardedPadding.
// Related: internal/agent/run/finish.go, internal/agent/run/journal.go.
package run

import (
	"strings"
	"testing"
	"unsafe"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestFinalResponseReleasesDiscardedPadding(t *testing.T) {
	raw := strings.Repeat(" ", 1<<20) + "answer\n"
	j := &journal{}
	s := execution{runner: fixture(&generator{}, &executor{}, j), emit: emit}
	if err := s.finish(t.Context(), modelapi.Response{Text: raw}); err != nil {
		t.Fatal(err)
	}
	if s.result.Text != "answer" || len(j.messages) != 1 || j.messages[0].Content != "answer" {
		t.Fatal("final answer changed")
	}
	if unsafe.StringData(s.result.Text) == unsafe.StringData(strings.TrimSpace(raw)) {
		t.Fatal("short answer retains the discarded large backing buffer")
	}
}
