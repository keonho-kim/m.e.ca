// Purpose: Verify single reviews, bounded corrections, referenced findings and cancellation.
// Pattern: Review use-case contract tests.
// Usage: go test ./internal/agent/conclave.
// Related: internal/agent/conclave/review.go.
package conclave

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func reviewInput(t *testing.T) Input {
	t.Helper()
	call := tool.Call{ID: "c", Name: "write_file", Arguments: []byte(`{"path":"a","content":"new"}`)}
	reason, err := grounding.Bind(call, "Implement requested value.", "r")
	if err != nil {
		t.Fatal(err)
	}
	return Input{Request: "replace old with new", Call: call, Grounding: reason, Candidate: tool.BindCandidate(call, []tool.FileDelta{{Path: "a", BeforeLines: 1, AfterLines: 1, Added: 1, Deleted: 1, Diff: "-old\n+new"}})}
}
func TestSingleReviewAndCorrectionBound(t *testing.T) {
	for _, replies := range [][]string{{`{"decision":"pass"}`}, {"bad", `{"decision":"pass"}`}, {"bad", "bad"}} {
		calls := 0
		result, err := Review(context.Background(), reviewInput(t), modelapi.Request{ModelID: "m"}, 65536, func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
			if len(r.Tools) != 0 {
				t.Fatal("review has executable tools")
			}
			if strings.Contains(r.Messages[len(r.Messages)-1].Content, `"Arguments"`) {
				t.Fatal("candidate duplicated or missing")
			}
			text := replies[calls]
			calls++
			return emit(modelapi.Event{Delta: text, Done: true})
		}, nil, func(s string) string { return s }, nil)
		wantPass := replies[len(replies)-1] != "bad"
		if err != nil || result.Passed != wantPass || calls != len(replies) {
			t.Fatalf("calls=%d result=%+v err=%v", calls, result, err)
		}
	}
}
func TestReviewRejectsInventedReferenceAndReadFailure(t *testing.T) {
	for _, reply := range []string{`{"decision":"reject","findings":[{"path":"other","side":"after","line":1,"reason":"wrong"}]}`, `{"decision":"needs_context","context_requests":[{"path":"outside","offset":0,"limit":100,"reason":"inspect"}]}`} {
		result, err := Review(context.Background(), reviewInput(t), modelapi.Request{}, 65536, func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
			return emit(modelapi.Event{Delta: reply, Done: true})
		}, func(context.Context, ContextRequest) (json.RawMessage, error) { return nil, fmt.Errorf("scope denied") }, func(s string) string { return s }, nil)
		if err != nil || result.Passed || result.Error == "" {
			t.Fatalf("%+v %v", result, err)
		}
	}
}

func TestPendingCandidateContextIsNotReadFromDisk(t *testing.T) {
	input := reviewInput(t)
	input.Pending = []tool.FileDelta{{Path: "implementation.py", Added: 1, AfterLines: 1, Diff: "+def f(): return 1"}}
	count := 0
	result, err := Review(context.Background(), input, modelapi.Request{}, 65536, func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		count++
		reply := `{"decision":"pass"}`
		if count == 1 {
			if !strings.Contains(r.Messages[len(r.Messages)-1].Content, "implementation.py") {
				t.Fatal("pending sibling not advertised")
			}
			reason := strings.Repeat("Read the pending implementation to compare the test. ", 8)
			body, _ := json.Marshal(Verdict{Decision: "needs_context", ContextRequests: []ContextRequest{{Path: "implementation.py", Limit: 8192, Reason: reason}}})
			reply = string(body)
		} else if !strings.Contains(r.Messages[len(r.Messages)-1].Content, `"executed":false`) {
			t.Fatal("proposed context lacks unexecuted label")
		}
		return emit(modelapi.Event{Delta: reply, Done: true})
	}, func(context.Context, ContextRequest) (json.RawMessage, error) {
		t.Fatal("read non-existent sibling from disk")
		return nil, nil
	}, func(s string) string { return s }, nil)
	if err != nil || !result.Passed || count != 2 {
		t.Fatalf("%+v %v", result, err)
	}
}
