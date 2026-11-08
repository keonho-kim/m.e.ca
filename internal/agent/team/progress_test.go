// Purpose: Recognize new shared work and execution evidence without renewing idle recovery on repeats.
// Pattern: Deterministic progress-classification regression tests.
// Usage: Team tests check request-owned bounded evidence independently of model output.
// Related: internal/agent/team/progress.go, internal/agent/team/idle.go.
package team

import (
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"encoding/json"
	"fmt"
	"strings"
	"testing"

	record "github.com/keonho-kim/meca/internal/agent/record"
	tool "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestInterfaceDecisionAdvancesProgressWithoutRewritingFiles(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	before := r.progress
	p.SetWork("Authentication interface", []string{"/project/auth"}, "Authenticate(token) returns User", 0)
	if r.progress == before {
		t.Fatal("new work and interface were not progress")
	}
	before = r.progress
	p.SetWork("Clarify wording", []string{"/project/auth"}, "Authenticate(token) returns User", 0)
	if r.progress != before {
		t.Fatal("summary paraphrasing renewed progress")
	}
	p.SetWork("Different interface", []string{"/project/auth"}, "Authenticate(token) returns Result", 0)
	if r.progress == before {
		t.Fatal("changed interface did not count")
	}
	before = r.progress
	p.SetWork("Original interface", []string{"/project/auth"}, "Authenticate(token) returns User", 0)
	if r.progress != before {
		t.Fatal("cycling back to a known contract renewed progress")
	}
}

func BenchmarkOperationEvidence(b *testing.B) {
	for _, size := range []int{0, 32 << 10, 1 << 20} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			output, _ := json.Marshal(map[string]string{"Stdout": strings.Repeat("x", size), "Stderr": ""})
			event := record.Event{Output: output, Operation: &tool.Operation{Name: "execute_command", Command: "check", Process: &tool.ProcessOutcome{Status: "exited"}}}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				operationProgress(event)
			}
		})
	}
}

func TestRepeatedOrEmptyCommandResultsDoNotRenewIdleRecovery(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	emit := func(id, stdout string, code int) {
		t.Helper()
		output, _ := json.Marshal(map[string]any{"Stdout": stdout, "Stderr": "", "command_id": id, "StdoutPath": "/logs/" + id, "elapsed_millis": len(id)})
		event := record.Event{Kind: "tool_succeeded", Output: output, Operation: &tool.Operation{Name: "execute_command", Status: "tool_succeeded", Command: "check", Process: &tool.ProcessOutcome{CommandID: id, Status: "exited", ExitCode: code}}}
		if err := p.Emit(event); err != nil {
			t.Fatal(err)
		}
	}
	emit("empty", "", 0)
	if r.progress != 0 {
		t.Fatal("empty success was counted as progress")
	}
	emit("first", "one test passed", 0)
	first := r.progress
	if first == 0 {
		t.Fatal("new observed output was not progress")
	}
	emit("new_id", "one test passed", 0)
	if r.progress != first {
		t.Fatal("volatile command metadata renewed identical evidence")
	}
	emit("changed", "two tests passed", 0)
	if r.progress == first {
		t.Fatal("changed evidence was ignored")
	}
}

func TestExplicitFailureIsEvidenceEvenWithEmptySuccessfulProcessOutput(t *testing.T) {
	event := record.Event{Operation: &tool.Operation{
		Name: "execute_command", Status: "tool_failed", Started: true, Command: "check",
		Process: &tool.ProcessOutcome{Status: "exited", ExitCode: 0},
		Failure: &faults.Failure{Code: "capture_failed", Detail: "Could not read stdout"},
	}}
	first, found := operationProgress(event)
	if !found {
		t.Fatal("an explicit failure was classified as an empty success")
	}
	event.Operation.Failure.Detail = "Could not read stderr"
	second, found := operationProgress(event)
	if !found || first == second {
		t.Fatal("different failure facts collapsed to the same observation")
	}
}

func TestRemovingAContractWithoutOwnedPathsAdvancesProgress(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	if _, err := p.SetWork("Shared result", nil, "UserID is a string", 0); err != nil {
		t.Fatal(err)
	}
	before := r.progress
	if _, err := p.SetWork("Withdraw the proposed contract", nil, "", 0); err != nil {
		t.Fatal(err)
	}
	if r.progress == before {
		t.Fatal("removing a shared contract was ignored after ownership release")
	}
}
