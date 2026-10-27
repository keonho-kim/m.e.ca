// Purpose: Verify outbound approval occurs before compound effects while local HTTP remains usable.
// Pattern: Filesystem and loopback HTTP integration tests.
// Usage: go test ./internal/tools/local -run TestOutboundCommand.
// Related: internal/tools/local/prepare.go, internal/policy/command/transfer.go.
package local

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/policy"
)

func TestOutboundCommandDenialPreventsCompoundEffects(t *testing.T) {
	e, project := testExecutor(t)
	text := `printf changed > marker && curl -s -X PUT --data-binary fixture 'https://upload.example.test/object?probe=fixture' | tail -3`
	for _, permission := range []policy.Permission{policy.PermissionDefault, policy.PermissionAllMighty} {
		op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System, Permission: permission}, call("execute_command", fmt.Sprintf(`{"command":%q}`, text)))
		if err != nil {
			t.Fatal(err)
		}
		approval := op.Approval()
		if !approval.Required || !approval.UserRequired || approval.Command == nil || len(approval.Command.Findings) != 1 {
			t.Fatalf("lost outbound review: %+v", approval)
		}
		finding := approval.Command.Findings[0]
		if finding.Command != "curl" || len(finding.Targets) != 1 || finding.Targets[0] != "https://upload.example.test/object" {
			t.Fatalf("unsafe or missing destination: %+v", finding)
		}
		// Approval enforcement belongs to the runtime in both modes. The local
		// execution adapter must never infer consent from All Mighty alone.
		if _, err := op.Execute(context.Background(), false); err == nil {
			t.Fatal("denied transfer executed")
		}
		if _, err := os.Stat(filepath.Join(project, "marker")); !os.IsNotExist(err) {
			t.Fatal("compound prefix executed before consent", err)
		}
	}
}

func TestOutboundCommandPreservesLocalHTTP(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl not installed")
	}
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || r.Method != "POST" || string(body) != "fixture" {
			t.Error("unexpected local request", err)
		}
		received.Add(1)
		fmt.Fprint(w, "local-ok")
	}))
	defer server.Close()
	e, _ := testExecutor(t)
	text := "curl -fsS -X POST --data-binary fixture " + server.URL + "/api"
	op, err := e.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, text)))
	if err != nil {
		t.Fatal(err)
	}
	if op.Approval().Required {
		t.Fatal("local API test unexpectedly needs approval")
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil || result.Failed || !strings.Contains(string(result.Content), "local-ok") || received.Load() != 1 {
		t.Fatalf("local request: %+v %v", result, err)
	}
}
