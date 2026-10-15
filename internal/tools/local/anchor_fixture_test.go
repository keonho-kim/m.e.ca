// Purpose: Obtain edit anchors through the real read_file boundary for file-tool tests.
// Pattern: Test fixture.
// Usage: File mutation tests call anchoredEdit with inclusive line numbers.
// Related: internal/tools/local/files_test.go, internal/tools/local/review_test.go, internal/tools/local/file/read.go.
package local

import (
	"context"
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"strconv"
	"strings"
	"testing"
)

func anchoredEdit(t *testing.T, e *Executor, path string, first, last int, replacement string) toolapi.Call {
	t.Helper()
	args, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		t.Fatal(err)
	}
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.Directory, Permission: policy.Approve}, call("read_file", string(args)))
	if err != nil {
		t.Fatal(err)
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	var read struct{ Content string }
	if err := json.Unmarshal(result.Content, &read); err != nil {
		t.Fatal(err)
	}
	var start, end string
	for _, line := range strings.Split(read.Content, "\n") {
		anchor, _, ok := strings.Cut(line, " | ")
		if !ok {
			continue
		}
		if strings.HasPrefix(anchor, strconv.Itoa(first)+"#") {
			start = anchor
		}
		if strings.HasPrefix(anchor, strconv.Itoa(last)+"#") {
			end = anchor
		}
	}
	if start == "" || end == "" {
		t.Fatalf("missing requested anchors: %s", read.Content)
	}
	args, err = json.Marshal(map[string]string{"path": path, "start_anchor": start, "end_anchor": end, "new_text": replacement})
	if err != nil {
		t.Fatal(err)
	}
	return call("edit_file", string(args))
}
