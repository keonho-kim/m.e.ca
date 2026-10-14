// Purpose: Verify file reversal and external changes in progress evidence.
// Pattern: State transition tests.
// Usage: go test ./internal/agent/progress.
// Related: internal/agent/progress/files.go.
package progress

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"testing"
)

func TestProgressDetectsContentReversal(t *testing.T) {
	s := &Files{}
	if s.Repeated([]toolapi.FileChange{{Path: "a", BeforeDigest: "first", AfterDigest: "second"}}) {
		t.Fatal("first change is not a cycle")
	}
	if !s.Repeated([]toolapi.FileChange{{Path: "a", BeforeDigest: "second", AfterDigest: "first"}}) {
		t.Fatal("reversal was counted as progress")
	}
	if s.Repeated([]toolapi.FileChange{{Path: "a", BeforeDigest: "external", AfterDigest: "fresh"}}) {
		t.Fatal("external state change ignored")
	}
}
