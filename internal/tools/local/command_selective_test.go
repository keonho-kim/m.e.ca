// Purpose: Verify selective command policy reaches the host shell and rejected compound commands have no effects.
// Pattern: Temporary-filesystem integration tests.
// Usage: go test ./internal/tools/local -run TestSelectiveCommand.
// Related: internal/tools/local/prepare.go, internal/tools/local/command_safety.go, internal/tools/local/operation.go.
package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func TestSelectiveCommandEditingPipeline(t *testing.T) {
	executor, project := testExecutor(t)
	suffix := ""
	if runtime.GOOS == "darwin" {
		suffix = " ''"
	}
	command := "printf 'old\\n' > source.txt && /usr/bin/sed -i" + suffix + " 's/old/new/g' source.txt && cat source.txt | /usr/bin/tr 'a-z' 'A-Z'"
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, command)))
	if err != nil {
		t.Fatal(err)
	}
	if op.Approval().Required {
		t.Fatalf("ordinary edit requires approval: %+v", op.Approval())
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil || result.Failed {
		t.Fatalf("%+v %v", result, err)
	}
	var output struct {
		Stdout   string
		ExitCode int
	}
	if err := json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Stdout != "NEW\n" || output.ExitCode != 0 {
		t.Fatalf("output=%+v", output)
	}
	if body, err := os.ReadFile(filepath.Join(project, "source.txt")); err != nil || string(body) != "new\n" {
		t.Fatalf("file=%q %v", body, err)
	}
}

func TestSelectiveCommandDenialPreventsEveryStage(t *testing.T) {
	executor, project := testExecutor(t)
	path := filepath.Join(t.TempDir(), "keep.txt")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	command := fmt.Sprintf("printf changed > marker && rm -f %q", path)
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, command)))
	if err != nil {
		t.Fatal(err)
	}
	approval := op.Approval()
	if !approval.Required || approval.Command == nil || len(approval.Command.Findings) != 1 || approval.Command.Findings[0].Command != "rm" {
		t.Fatalf("review did not isolate rm: %+v", approval.Command)
	}
	if _, err := op.Execute(context.Background(), false); err == nil {
		t.Fatal("unapproved compound executed")
	}
	if _, err := os.Stat(filepath.Join(project, "marker")); !os.IsNotExist(err) {
		t.Fatalf("earlier command ran: %v", err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "keep" {
		t.Fatalf("denied file changed: %q %v", body, err)
	}
}

func TestSelectiveCommandUnregisteredExecutablesUseDefault(t *testing.T) {
	executor, _ := testExecutor(t)
	for _, command := range []string{`unregistered-one sync && unregistered-two install stable`, `env RESULT_PATH="$(unregistered-command locate)" unregistered-command run task`, `unregistered-command run nested-command --target "$DYNAMIC_TARGET"`} {
		op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, command)))
		if err != nil {
			t.Fatal(err)
		}
		if op.Approval().Required {
			t.Fatalf("unregistered executable was inspected: %+v", op.Approval())
		}
	}
}

func TestSelectiveCommandErrorsIdentifyTheirCause(t *testing.T) {
	executor, _ := testExecutor(t)
	for _, test := range []struct{ command, code string }{{"if then", toolapi.InvalidCommand}, {"rm -rf /etc", toolapi.CommandBlocked}} {
		_, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, test.command)))
		var failure *toolapi.Failure
		if !errors.As(err, &failure) || failure.Code != test.code {
			t.Fatalf("command=%q error=%v", test.command, err)
		}
	}
}

func TestSelectiveCommandReviewRedactsEnvironment(t *testing.T) {
	executor, _ := testExecutor(t)
	executor.redact = func(text string) string { return strings.ReplaceAll(text, "private-token", "[redacted]") }
	t.Setenv("V_REVIEW_TARGET", filepath.Join(t.TempDir(), "private-token", "file"))
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", `{"command":"printf x > \"$V_REVIEW_TARGET\""}`))
	if err != nil {
		t.Fatal(err)
	}
	proposal := op.Approval()
	if !proposal.Required {
		t.Fatal("outside write was not reviewed")
	}
	encoded, err := json.Marshal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private-token") || !strings.Contains(string(encoded), "[redacted]") {
		t.Fatal("expanded environment target leaked into approval")
	}
}

func TestSelectiveCommandPipelineStatusNeedsNoApproval(t *testing.T) {
	executor, project := testExecutor(t)
	if err := os.Mkdir(filepath.Join(project, "child"), 0700); err != nil {
		t.Fatal(err)
	}
	command := `cd child && printf 'test output\n' 2>&1 | tail -n 40; echo "EXIT: $?"`
	op, err := executor.Prepare(context.Background(), policy.Settings{Scope: policy.System}, call("execute_command", fmt.Sprintf(`{"command":%q}`, command)))
	if err != nil {
		t.Fatal(err)
	}
	if op.Approval().Required {
		t.Fatalf("fixed pipeline commands required approval: %+v", op.Approval())
	}
	result, err := op.Execute(context.Background(), false)
	if err != nil || result.Failed {
		t.Fatalf("pipeline did not execute: %+v %v", result, err)
	}
	var output struct{ Stdout string }
	if err := json.Unmarshal(result.Content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Stdout != "test output\nEXIT: 0\n" {
		t.Fatalf("unexpected output: %+v", output)
	}
}
