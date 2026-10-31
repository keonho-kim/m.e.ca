// Purpose: Verify exact final reports and actual command outcomes are preserved.
// Pattern: Scripted execution contract tests.
// Usage: go test ./internal/agent/run -run TestReport.
// Related: internal/agent/run/finish.go, internal/agent/run/observations.go.
package run

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestReportPreservesEntireFinalAnswer(t *testing.T) {
	for _, test := range []struct {
		name, request, tool, report string
		arguments                   map[string]any
	}{
		{"file change", "create README", "write_file", "완료한 변경: README 작성\n실제 확인: 없음\n미검증: 실행 테스트", map[string]any{"path": "README.md", "content": "hello"}},
		{"long inspection report", "read a", "inspect_fixture", strings.Repeat("Observed file contents. ", 500) + "No deployment was performed.", map[string]any{"path": "a"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := &generator{replies: []string{nativeCall(test.tool, test.arguments), test.report}}
			e, j := &executor{}, &journal{}
			result, err := fixture(g, e, j).Run(context.Background(), test.request, approve, emit)
			if err != nil || result.Status != runrecord.Answered || result.Text != test.report || len(g.requests) != 2 || e.executed != 1 {
				t.Fatalf("result=%+v requests=%d executed=%d err=%v", result, len(g.requests), e.executed, err)
			}
			if last := j.messages[len(j.messages)-1]; last.Role != "assistant" || last.Content != test.report {
				t.Fatal("journal did not preserve the complete final report")
			}
		})
	}
}

func TestReportRetainsFailedCommandFacts(t *testing.T) {
	report := "변경 완료. 실제 확인: test 실패. 미검증: 정상 동작."
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), nativeCall("execute_command", map[string]any{"command": "test-a"}), report}}
	e := &executor{outcomes: []toolapi.Result{{Content: []byte(`{"changed":true}`)}, {Failed: true, Content: []byte(`{"stderr":"assertion failed","exit_code":7}`), Process: &toolapi.ProcessOutcome{Status: "exited", ExitCode: 7}}}}
	j := &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "edit and test", approve, emit)
	if err != nil || result.Text != report || len(g.requests) != 3 || result.Status != runrecord.Answered || !result.HadToolFailure {
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

func TestRepeatedCommandIsADeliberateNewExecution(t *testing.T) {
	g := &generator{replies: []string{
		`native:{"id":"first","name":"execute_command","arguments":{"command":"check-service"}}`,
		`native:{"id":"second","name":"execute_command","arguments":{"command":"check-service"}}`,
		"Service is ready.",
	}}
	e := &executor{}
	result, err := fixture(g, e, &journal{}).Run(context.Background(), "Wait for the service", approve, emit)
	if err != nil || result.Status != runrecord.Answered || e.executed != 2 || len(g.requests) != 3 || result.Text != "Service is ready." {
		t.Fatalf("result=%+v executed=%d requests=%d err=%v", result, e.executed, len(g.requests), err)
	}
}
