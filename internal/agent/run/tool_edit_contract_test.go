// Purpose: Verify read-then-edit works through native calls with Permission as the approval authority.
// Pattern: Scripted filesystem integration test.
// Usage: go test ./internal/agent/run -run TestToolReadThenEdit.
// Related: internal/tools/local/prepare.go, internal/agent/run/action_result.go.
package run

import (
	"context"
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolReadThenEditUsesOnlyPermission(t *testing.T) {
	for _, permission := range []policy.Permission{policy.Approve, policy.AllowAll} {
		t.Run(string(permission), func(t *testing.T) {
			r, _, root, _, _ := realBatchRunner(t, nil)
			r.Settings.Permission = permission
			path := filepath.Join(root, "note.txt")
			if err := os.WriteFile(path, []byte("name: demo\nstatus: pending\ncount: 3\n"), 0600); err != nil {
				t.Fatal(err)
			}
			g := &generator{replies: []string{nativeCall("execute_command", map[string]any{"command": "cat note.txt"}), "missing read result", "Updated the requested second line."}}
			r.Generator = g
			approvals := 0
			result, err := r.Run(context.Background(), "Read note.txt and change only status: pending to status: done.", func(_ context.Context, proposal toolapi.Approval) (bool, error) {
				approvals++
				if proposal.Candidate != nil && proposal.Candidate.Lines() != 2 {
					t.Fatal("approval not bound to actual edit")
				}
				return true, nil
			}, func(e Event) error {
				if e.Kind == "tool_succeeded" && e.Call != nil && e.Call.Name == "execute_command" {
					var read struct{ Stdout string }
					if err := json.Unmarshal(e.Output, &read); err != nil {
						return err
					}
					old := strings.SplitAfter(read.Stdout, "\n")[1]
					g.replies[1] = nativeCall("edit_file", map[string]any{"path": "note.txt", "edits": []map[string]any{{"old_text": old, "new_text": "status: done"}}})
				}
				return nil
			})
			if err != nil || result.Status != Answered {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != "name: demo\nstatus: done\ncount: 3\n" {
				t.Fatalf("content=%q error=%v", got, err)
			}
			want := 0
			if permission == policy.Approve {
				want = 2
			}
			if approvals != want {
				t.Fatalf("approvals=%d want=%d", approvals, want)
			}
		})
	}
}
