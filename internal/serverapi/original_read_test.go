// Purpose: Recover accepted originals larger than preview budgets and retain external Responses history across continuations.
// Pattern: Real engine and SQLite API contract tests.
// Usage: go test ./internal/serverapi -run 'TestLargeOriginal|TestResponseExternal'.
// Related: internal/commands/server/response.go, internal/commands/server/submission_session.go.
package serverapi

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/commands/server"
)

func TestLargeOriginalSurvivesSmallerPreviewBudget(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	if _, err := f.app.Select(t.Context(), "", true); err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("x", server.MaxReadBytes+1024)
	id := f.store.Session().ID
	before, err := f.store.OutputPosition(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.AppendMessage(t.Context(), id, model.Message{Role: "assistant", Content: text}); err != nil {
		t.Fatal(err)
	}
	through, err := f.store.OutputPosition(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.app.ResponseResult(t.Context(), server.Response{Session: id, Begin: before, Through: through})
	if err != nil || got != text {
		t.Fatal("accepted original cannot be recovered", len(got), err)
	}
}

func TestResponseExternalHistorySurvivesContinuation(t *testing.T) {
	requests := make(chan model.Request, 2)
	f := newFixture(t, generateFunc(func(_ context.Context, request model.Request, emit func(model.Event) error) error {
		requests <- request
		return emit(model.Event{Delta: "done", Done: true})
	}), false)
	first, err := f.app.Submit(t.Context(), server.Submission{Kind: "response", Stored: true, Messages: []server.InputMessage{{Role: "user", Text: "external-history-canary"}, {Role: "assistant", Text: "client answer"}, {Role: "user", Text: "first actual input"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.WaitResponse(t.Context(), first.Response.ID); err != nil {
		t.Fatal(err)
	}
	<-requests
	second, err := f.app.Submit(t.Context(), server.Submission{Kind: "response", Stored: true, Previous: first.Response.ID, Messages: []server.InputMessage{{Role: "user", Text: "continue"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.WaitResponse(t.Context(), second.Response.ID); err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	for _, m := range (<-requests).Messages {
		body.WriteString(m.Content)
	}
	if strings.Count(body.String(), "external-history-canary") != 1 {
		t.Fatal("continuation lost or duplicated initial external context")
	}
}
