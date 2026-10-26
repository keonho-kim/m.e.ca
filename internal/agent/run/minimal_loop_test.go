// Purpose: Verify ordinary native calls and deliberate repeated commands use one direct loop.
// Pattern: Behavioral regression tests.
// Usage: go test ./internal/agent/run -run TestMinimal.
// Related: internal/agent/run/run.go, internal/agent/run/call_validation.go.
package run

import (
	"context"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"strings"
	"testing"
)

func TestMinimalRepeatedCommandNeedsNoReasonOrInterveningEdit(t *testing.T) {
	g := &generator{replies: []string{
		`native:{"id":"first","name":"execute_command","arguments":{"command":"check-service"}}`,
		`native:{"id":"second","name":"execute_command","arguments":{"command":"check-service"}}`,
		"Service is ready.",
	}}
	e := &executor{}
	r := fixture(g, e, &journal{})
	result, err := r.Run(context.Background(), "Wait for the service", approve, emit)
	if err != nil || result.Status != runrecord.Answered || e.executed != 2 {
		t.Fatalf("result=%+v executed=%d err=%v", result, e.executed, err)
	}
}

func TestMinimalEmptyResponseStopsOnce(t *testing.T) {
	g := &generator{replies: []string{""}}
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "Hello", approve, emit)
	if err == nil || !strings.Contains(err.Error(), "empty") || result.Status != runrecord.Stopped || len(g.requests) != 1 {
		t.Fatalf("%+v calls=%d err=%v", result, len(g.requests), err)
	}
}

func TestMinimalJSONAnswerIsOrdinaryText(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"example-only"}`}}
	e := &executor{}
	result, err := fixture(g, e, &journal{}).Run(context.Background(), "Show an example JSON object", approve, emit)
	if err != nil || result.Status != runrecord.Answered || e.executed != 0 {
		t.Fatalf("%+v err=%v", result, err)
	}
}
