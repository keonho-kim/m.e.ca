// Purpose: Verify final reports survive without a runtime verification gate.
// Pattern: Scripted execution contract tests.
// Usage: go test ./internal/agent/run -run TestReport.
// Related: internal/agent/run/finish.go, internal/agent/run/observation.go.
package run

import (
	"context"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestReportAfterChangeWithoutCheck(t *testing.T) {
	report := "완료한 변경: README 작성\n실제 확인: 없음\n미검증: 실행 테스트"
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "README.md", "content": "hello"}), report}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "create README", approve, emit)
	if err != nil || result.Status != Answered || result.Text != report || len(g.requests) != 2 {
		t.Fatalf("report replaced or extra generation: %+v requests=%d err=%v", result, len(g.requests), err)
	}
}

func TestReportRetainsFailedCommandFacts(t *testing.T) {
	report := "변경 완료. 실제 확인: test 실패. 미검증: 정상 동작."
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), nativeCall("execute_command", map[string]any{"command": "test-a"}), report, report}}
	e := &executor{outcomes: []toolapi.Result{{Content: []byte(`{"changed":true}`)}, {Failed: true, Content: []byte(`{"stderr":"assertion failed","exit_code":7}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 7}}}}
	j := &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "edit and test", approve, emit)
	if err != nil || !strings.Contains(result.Text, "test") || result.Status != Stopped || !result.HadToolFailure {
		t.Fatalf("%+v %v", result, err)
	}
	found := false
	for _, event := range j.events {
		if event.Operation != nil && event.Operation.Process != nil && event.Operation.Process.ExitCode == 7 && strings.Contains(string(event.Output), "assertion failed") {
			found = true
		}
	}
	if !found {
		t.Fatal("actual command outcome was lost")
	}
}

func TestRepeatedCommandUsesRecordedOutcome(t *testing.T) {
	command := nativeCall("execute_command", map[string]any{"command": "test-a"})
	g, e := &generator{replies: []string{command, command, "The command already ran"}}, &executor{}
	j := &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "run once", approve, emit)
	if err != nil || result.Status != Answered || e.executed != 1 {
		t.Fatalf("%+v executions=%d %v", result, e.executed, err)
	}
	found := false
	for _, message := range j.messages {
		if message.Role == "tool" && strings.Contains(message.Content, "Repeated identical command") {
			found = true
		}
	}
	if !found {
		t.Fatal("missing duplicate feedback")
	}
}
