// Purpose: Recognize a return to earlier file content within one request.
// Pattern: Request-progress state transition regression.
// Usage: go test ./internal/agent/run -run TestRequestFileCycle.
// Related: internal/agent/run/semantic_progress.go, internal/agent/progress/request.go.
package run

import (
	"testing"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func TestRequestFileCycleResetsForNewRequest(t *testing.T) {
	first := execution{}
	call := toolapi.Call{Name: "edit_file"}
	first.toolAdvanced(call, toolapi.Result{Changes: []toolapi.FileChange{{Path: "/main.py", BeforeDigest: "original", AfterDigest: "corrected"}}}, policy.Change, "")
	if first.round.cycle {
		t.Fatal("first change marked as a cycle")
	}
	next := &first
	next.toolAdvanced(call, toolapi.Result{Changes: []toolapi.FileChange{{Path: "/main.py", BeforeDigest: "corrected", AfterDigest: "original"}}}, policy.Change, "")
	if !next.round.cycle {
		t.Fatal("request lost the return to earlier source")
	}
	fresh := execution{}
	fresh.toolAdvanced(call, toolapi.Result{Changes: []toolapi.FileChange{{Path: "/main.py", BeforeDigest: "corrected", AfterDigest: "original"}}}, policy.Change, "")
	if fresh.round.cycle {
		t.Fatal("a distinct user request inherited another request's cycle")
	}
}
