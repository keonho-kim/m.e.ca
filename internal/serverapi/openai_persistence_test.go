// Purpose: Verify compatible tasks have one durable authority and remain visible through native lookup.
// Pattern: Cross-adapter lifecycle regression test.
// Usage: go test ./internal/serverapi -run TestOpenAIUsesOneDurableRecord.
// Related: internal/commands/server/submission.go, internal/commands/server/tasks.go.
package serverapi

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/storage/serverrequests"
)

func TestOpenAIUsesOneDurableRecord(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := openAITransport(t, f, "sse")
	reply := openAIRequest(t, transport, "POST", "/v1/responses", `{"model":"meca","input":"hello"}`, "one-authority")
	if reply.Code != 200 {
		t.Fatal(reply.Code, reply.Body.String())
	}
	id := responseID(t, reply.Body.Bytes())
	if f.outputReads.Load() != 1 || f.resultOpens.Load() != 0 {
		t.Fatal("selected result reopened its database", f.outputReads.Load(), f.resultOpens.Load())
	}
	taskID := reply.Header().Get("X-Meca-Task-ID")
	db, err := sql.Open("sqlite", filepath.Join(f.requests, server.ProjectScope(f.project), serverrequests.Filename))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	var digest string
	if err := db.QueryRow("SELECT count(*),hex(id) FROM requests").Scan(&count, &digest); err != nil || count != 1 || strings.ToLower(digest) != id[5:] || taskID[5:] != id[5:] {
		t.Fatal("task and response have different durable authorities", count, digest, err)
	}
	// Remove the selected conversation and last-task snapshot. Native lookup
	// must reconstruct the same task from its response record.
	if err := f.app.CloseSession(); err != nil {
		t.Fatal(err)
	}
	task, err := f.app.TaskStatus(t.Context(), taskID)
	if err != nil || task.ID != taskID || task.Status != "answered" {
		t.Fatal("native result lost after detaching session", task, err)
	}
	lookup := openAIRequest(t, transport, "GET", "/v1/responses/"+id, "", "")
	if lookup.Code != 200 || lookup.Body.String() != reply.Body.String() || f.resultOpens.Load() != 1 {
		t.Fatal("detached result did not use its independent reader", lookup.Code, f.resultOpens.Load())
	}
}
