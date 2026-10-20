// Purpose: Verify creation receipts distinguish successful directory and zero-byte file creation.
// Pattern: Tool result contract tests.
// Usage: go test ./internal/tools/local -run TestCreationReceipt.
// Related: internal/tools/local/mutations.go.
package local

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
)

func TestCreationReceipt(t *testing.T) {
	e, _ := testExecutor(t)
	settings := policy.Settings{Scope: policy.System}
	for _, tc := range []struct {
		name, args, want string
		size             bool
	}{
		{"create_directory", `{"paths":["project"]}`, "Created directories: 1", false},
		{"write_file", `{"path":"project/empty","content":""}`, "(0 bytes)", true},
	} {
		op, err := e.Prepare(context.Background(), settings, call(tc.name, tc.args))
		if err != nil {
			t.Fatal(err)
		}
		result, err := op.Execute(context.Background(), true)
		if err != nil || result.Failed || !strings.Contains(result.Summary, tc.want) {
			t.Fatalf("%+v %v", result, err)
		}
		var receipt struct {
			Action string
			Size   *int `json:"size_bytes"`
		}
		if err := json.Unmarshal(result.Content, &receipt); err != nil {
			t.Fatal(err)
		}
		if receipt.Action != tc.name || tc.size && (receipt.Size == nil || *receipt.Size != 0) {
			t.Fatalf("ambiguous creation receipt: %s", result.Content)
		}
	}
}
