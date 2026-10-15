// Purpose: Verify post-result completion decisions control work, scope, and repeat exits.
// Pattern: Scripted model integration tests.
// Usage: go test ./internal/agent/run -run TestCompletionReview.
// Related: internal/agent/run/completion_review.go, internal/agent/completion/review.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent/completion"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func TestCompletionReviewContinuesOnlyMissingWork(t *testing.T) {
	g := &generator{replies: []string{
		nativeCall("write_file", map[string]any{"path": "a", "content": ""}),
		"Both files created",
		nativeCall("write_file", map[string]any{"path": "b", "content": ""}),
		"Created a and b; no tests needed for empty files.",
	}}
	g.completion = func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		if len(r.Tools) > 0 {
			t.Fatal("review received executable tools")
		}
		text := r.Messages[len(r.Messages)-1].Content
		var packet completion.Input
		if err := json.Unmarshal([]byte(text[strings.Index(text, "\n")+1:]), &packet); err != nil {
			t.Fatal(err)
		}
		if len(g.completionRequests) == 1 {
			return emit(modelapi.Event{Delta: `{"decision":"continue","reason":"b has not been created","unmet":["Create b"],"evidence":["request","call_1"]}`, Done: true})
		}
		if len(packet.State.Receipts) != 2 || packet.State.Receipts[0].Stale {
			t.Fatal("unrelated creation lost earlier evidence")
		}
		return emit(modelapi.Event{Delta: `{"decision":"complete","reason":"Both files created","evidence":["call_1","call_3"]}`, Done: true})
	}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "create a and b", approve, emit)
	if err != nil || result.Status != Answered || e.executed != 2 || len(g.completionRequests) != 2 || len(g.requests) != 4 {
		t.Fatalf("%+v %v", result, err)
	}
	for _, m := range j.messages {
		if m.Role == "assistant" && m.Content == "Both files created" {
			t.Fatal("unapproved draft committed as final")
		}
	}
	if !strings.Contains(g.requests[2].Messages[len(g.requests[2].Messages)-1].Content, "b has not been created") {
		t.Fatal("missing bounded continuation feedback")
	}
}
func TestCompletionReviewFailureBlocksAndReports(t *testing.T) {
	g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a"}), "Done", "Could not establish completion; the read succeeded."}}
	g.completion = func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
		return errors.New("review unavailable")
	}
	e := &executor{}
	result, err := fixture(g, e, &journal{}).Run(context.Background(), "inspect a", approve, emit)
	if err != nil || result.Status != Stopped || e.executed != 1 || len(g.completionRequests) != 1 || len(g.requests) != 2 || !strings.Contains(result.Text, "review unavailable") {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestCompletionReviewSameStateIsNotReviewedAgain(t *testing.T) {
	g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a"}), "Done", "Done again", "Read a; unresolved requirements remain."}}
	g.completion = func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		return emit(modelapi.Event{Delta: `{"decision":"continue","reason":"Required inspection remains","unmet":["Finish the requested inspection"],"evidence":["request"]}`, Done: true})
	}
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "inspect", approve, emit)
	if err != nil || result.Status != Stopped || len(g.completionRequests) != 1 {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestCompletionReviewOrdinaryChatNeedsNoJudge(t *testing.T) {
	g := &generator{replies: []string{"Hello"}}
	if _, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "hello", approve, emit); err != nil {
		t.Fatal(err)
	}
	if len(g.completionRequests) != 0 {
		t.Fatal("chat invoked reviewer")
	}
}
func TestCompletionReviewUnexpectedMetadataBlocksWholeBatch(t *testing.T) {
	calls := []toolapi.Call{
		batchCall("a", "read_file", map[string]any{"path": "a", "access": "read_only"}),
		batchCall("b", "write_file", map[string]any{"path": "b", "content": ""}),
	}
	r, _, _, j, _ := realBatchRunner(t, calls)
	if _, err := r.Run(context.Background(), "analysis only", approve, emit); err != nil {
		t.Fatal(err)
	}
	for _, e := range j.events {
		if e.Kind == "tool_started" {
			t.Fatal("invalid metadata batch executed a subset")
		}
	}
}
func TestCompletionReviewLegacyCompletesRejected(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "", "completes": []string{"create a"}}), "Invalid metadata; no changes."}}
	e := &executor{}
	if _, err := fixture(g, e, &journal{}).Run(context.Background(), "create a", approve, emit); err != nil {
		t.Fatal(err)
	}
	if e.executed != 0 {
		t.Fatal("obsolete pre-execution completion accepted")
	}
}
