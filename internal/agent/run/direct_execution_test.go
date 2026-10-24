// Purpose: Keep file execution and final reporting independent of semantic model reviews.
// Pattern: Runtime boundary regression tests.
// Usage: go test ./internal/agent/run -run 'TestDirectExecution|TestDeniedChange'.
// Related: internal/agent/run/prepare_execution.go, internal/agent/run/finish.go.
package run

import (
	"context"
	"errors"
	"fmt"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type directGenerator struct{ generator }

func (g *directGenerator) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	if len(request.Tools) == 0 {
		return fmt.Errorf("unexpected separate judgement request")
	}
	return g.generator.Generate(ctx, request, emit)
}

func TestDirectExecutionNeedsOnlyWorkerCalls(t *testing.T) {
	g := &directGenerator{generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), "Changed a; tests were not run."}}}
	e, j := &executor{approval: true}, &journal{}
	r := fixture(&g.generator, e, j)
	r.Generator = g
	approvals := 0
	result, err := r.Run(context.Background(), "Change a", func(context.Context, toolapi.Approval) (bool, error) {
		approvals++
		return true, nil
	}, emit)
	if err != nil || result.Status != Answered || e.executed != 1 || approvals != 1 || len(g.requests) != 2 {
		t.Fatalf("result=%+v executed=%d approvals=%d calls=%d err=%v", result, e.executed, approvals, len(g.requests), err)
	}
}

func TestDeniedChangeStillExecutesNothing(t *testing.T) {
	g := &directGenerator{generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"})}}}
	e := &executor{approval: true}
	r := fixture(&g.generator, e, &journal{})
	r.Generator = g
	result, err := r.Run(context.Background(), "Change a", func(context.Context, toolapi.Approval) (bool, error) { return false, nil }, emit)
	if !errors.Is(err, errApprovalDenied) || result.Status != Stopped || e.executed != 0 || len(g.requests) != 1 {
		t.Fatalf("result=%+v executed=%d calls=%d err=%v", result, e.executed, len(g.requests), err)
	}
}
