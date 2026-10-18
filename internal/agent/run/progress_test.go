// Purpose: Verify file reversals and repeated observations trigger bounded termination without replaying mutations.
// Pattern: Runtime progress state tests.
// Usage: go test ./internal/agent/run -run TestProgress.
// Related: internal/agent/run/progress.go, internal/agent/run/action.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
)

func TestProgressFreshVerificationIsNotARepeat(t *testing.T) {
	g := &generator{replies: []string{nativeCall("execute_command", map[string]any{"command": "verify"}), nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), nativeCall("execute_command", map[string]any{"command": "verify"}), "Verified"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	r.Settings.Scope = policy.System
	r.Limits = DefaultLimits()
	r.Limits.StallIterations = 1
	result, err := r.Run(context.Background(), "change and verify", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("result=%+v err=%v", result, err)
	}

}

func TestProgressEmptyRepliesEndWithStoppedReport(t *testing.T) {
	g := &generator{replies: []string{"", "", "", "Cannot determine a useful action."}}
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "inspect", approve, emit)
	if err != nil || result.Status != Stopped || result.Text == "" {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestProgressAlternatingCommandChecksAreNotNewChanges(t *testing.T) {
	var replies []string
	for _, command := range []string{"ls", "find .", "ls", "find .", "ls"} {
		replies = append(replies, nativeCall("execute_command", map[string]any{"command": command}))
	}
	replies = append(replies, "The requested files already exist; reporting the result.")
	g, j := &generator{replies: replies}, &journal{}
	result, err := fixture(g, &executor{}, j).Run(context.Background(), "create only empty files", approve, emit)
	if err != nil || result.Status != Stopped {
		t.Fatalf("%+v %v", result, err)
	}
	if len(g.requests) > 5 || !strings.Contains(result.Text, "progress") {
		t.Fatal("alternating unchanged commands did not stop")
	}

}
