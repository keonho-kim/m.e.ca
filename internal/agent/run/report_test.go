// Purpose: Verify report-only exits respect budgets, cancellation and persistence failures.
// Pattern: Scripted terminal transition tests.
// Usage: go test ./internal/agent/run -run TestFinalReport.
// Related: internal/agent/run/report.go, internal/agent/run/run.go.
package run

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFinalReportDoesNotSpendAnotherCallAtBudget(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "a"}), "Read a; the remaining work was not performed."}}
	r := fixture(g, &executor{}, &journal{})
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 2
	result, err := r.Run(context.Background(), "inspect a and b", approve, emit)
	if err != nil || result.Status != Answered || len(g.requests) != 2 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestFinalReportFallbackAtExhaustedBudget(t *testing.T) {
	g := &generator{replies: []string{nativeCall("inspect_fixture", map[string]any{"path": "a"})}}
	r := fixture(g, &executor{}, &journal{})
	r.Limits = DefaultLimits()
	r.Limits.MaxModelCalls = 1
	result, err := r.Run(context.Background(), "inspect a and b", approve, emit)
	if err != nil || result.Status != Stopped || len(g.requests) != 1 || !strings.Contains(result.Text, "마지막 관찰") {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestFinalReportCancellationDoesNotGenerateFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": ""}), "partial report"}, cancelAt: 2, cancel: cancel}
	result, err := fixture(g, &executor{}, &journal{}).Run(ctx, "create a", approve, emit)
	if !errors.Is(err, context.Canceled) || result.Status != Interrupted {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestFinalReportPreservesRuntimeDeliveryFailure(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": ""}), "report"}}
	sentinel := errors.New("UI delivery failed")
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "create a", approve, func(e Event) error {
		if e.Kind == "stream" && len(g.requests) == 1 {
			return sentinel
		}
		return nil
	})
	if !errors.Is(err, sentinel) || result.Status != Stopped {
		t.Fatalf("hidden runtime error: %+v %v", result, err)
	}
}
