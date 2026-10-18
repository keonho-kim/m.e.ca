// Purpose: Verify ordinary commands retain unchanged file facts and invalidate actual changes.
// Pattern: Filesystem integration regressions with scripted model responses.
// Usage: go test ./internal/agent/run -run TestFileFacts.
// Related: internal/agent/run/file_facts.go, internal/tools/local/versions.go.
package run

import (
	"context"
	"testing"
)

func TestFileFactsCompareActualVersionsAfterCommands(t *testing.T) {
	for _, tc := range []struct {
		command string
		stale   bool
	}{{"ls", false}, {"perl -e 'open(my $f, q(>), q(a)) or die $!; print $f q(changed)'", true}} {
		t.Run(tc.command, func(t *testing.T) {
			r, _, _, j, _ := realBatchRunner(t, nil)
			g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "initial"}), nativeCall("execute_command", map[string]any{"command": tc.command}), "Observed the files and command output."}}
			r.Generator = g
			result, err := r.Run(context.Background(), "Create a, then perform the specified command", approve, emit)
			if err != nil || result.Status != Answered {
				t.Fatalf("%+v %v", result, err)
			}
			var latest *Checkpoint
			for _, event := range j.events {
				if event.Checkpoint != nil {
					latest = event.Checkpoint
				}
			}
			if latest == nil || len(latest.Work.Receipts) != 2 || latest.Work.Receipts[0].Stale != tc.stale {
				t.Fatalf("lost file freshness: %+v", latest)
			}

		})
	}
}
