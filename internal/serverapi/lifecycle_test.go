// Purpose: Verify request ownership, approval binding, steering and resource release through the real runtime.
// Pattern: Server use case integration tests.
// Usage: go test ./internal/serverapi -run 'TestRequest|TestApproval|TestShutdown'.
// Related: internal/serverapi/fixture_test.go, internal/commands/server/controls.go.
package serverapi

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestRequestSurvivesObserverAndSubmissionCancellation(t *testing.T) {
	ctx := deadline(t)
	release := make(chan struct{})
	started := make(chan struct{})
	var calls atomic.Int32
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
		if calls.Add(1) > 1 {
			return emit(model.Event{Delta: "steered answer", Done: true})
		}
		if err := emit(model.Event{Delta: "early"}); err != nil {
			return err
		}
		close(started)
		select {
		case <-release:
			return emit(model.Event{Delta: " answer", Done: true})
		case <-ctx.Done():
			return ctx.Err()
		}
	}), false)
	state, err := f.app.Select(ctx, "", true)
	if err != nil {
		t.Fatal(err)
	}
	submit, cancel := context.WithCancel(ctx)
	task, err := f.app.Start(submit, state.Session.ID, "request-1", "answer")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	observer, stop := context.WithCancel(ctx)
	stop()
	if _, err := f.app.Next(observer, f.app.State().Cursor); err != context.Canceled {
		t.Fatal(err)
	}
	if f.app.State().Task.Status != "running" {
		t.Fatal("connection canceled the task")
	}
	again, err := f.app.Start(ctx, state.Session.ID, "request-1", "answer")
	if err != nil || again.ID != task.ID {
		t.Fatal(again, err)
	}
	if _, err := f.app.Start(ctx, state.Session.ID, "request-1", "changed"); err == nil {
		t.Fatal("changed retry accepted")
	}
	if _, err := f.app.Start(ctx, state.Session.ID, "request-2", "answer"); err == nil {
		t.Fatal("concurrent request accepted")
	}
	if err := f.app.CloseSession(t.Context(), f.app.State().Session.ID); err == nil {
		t.Fatal("active session closed")
	}
	if err := f.app.Steer(task.ID, "finish with the new instruction"); err != nil {
		t.Fatal(err)
	}
	cursor := f.app.State().Cursor
	close(release)
	finished := waitFinished(t, ctx, f.app, cursor)
	if finished.ID != task.ID || finished.Status != "answered" || calls.Load() != 2 {
		t.Fatal(finished, calls.Load())
	}
	items, _, err := f.app.History(ctx, "", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Message.Role == "user" && item.Message.Content == "finish with the new instruction" {
			found = true
		}
	}
	if !found {
		t.Fatal("steering was not admitted into the conversation")
	}
}

func TestApprovalIsSingleUseAndHistoryRetainsToolOriginal(t *testing.T) {
	ctx := deadline(t)
	var calls atomic.Int32
	f := newFixture(t, generateFunc(func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
		if calls.Add(1) == 1 {
			return emit(model.Event{Calls: []tool.Call{{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"path":"result.txt","content":"written"}`)}}, Done: true})
		}
		return emit(model.Event{Delta: "done", Done: true})
	}), true)
	state, err := f.app.Select(ctx, "", true)
	if err != nil {
		t.Fatal(err)
	}
	task, err := f.app.Start(ctx, state.Session.ID, "request-approve", "write a file")
	if err != nil {
		t.Fatal(err)
	}
	cursor := state.Cursor
	approval := ""
	for approval == "" {
		batch, err := f.app.Next(ctx, cursor)
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range batch.Events {
			cursor = event.Cursor
			if event.Record.Kind == "approval_required" {
				approval = event.ApprovalID
			}
		}
	}
	if _, err := os.Stat(filepath.Join(f.project, "result.txt")); !os.IsNotExist(err) {
		t.Fatal("effect preceded approval", err)
	}
	if err := f.app.Approve("wrong-task", approval, true); err == nil {
		t.Fatal("wrong task approval accepted")
	}
	if err := f.app.Approve(task.ID, approval, true); err != nil {
		t.Fatal(err)
	}
	if err := f.app.Approve(task.ID, approval, true); err == nil {
		t.Fatal("repeated approval accepted")
	}
	finished := waitFinished(t, ctx, f.app, cursor)
	if finished.Status != "answered" || f.executed.Load() != 1 {
		t.Fatal(finished, f.executed.Load())
	}
	body, err := os.ReadFile(filepath.Join(f.project, "result.txt"))
	if err != nil || string(body) != "written" {
		t.Fatal(string(body), err)
	}
	items, _, err := f.app.History(ctx, "", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Event.Reference != nil {
			page, err := f.api.readDetail(ctx, &v1.DetailRequest{Source: &v1.DetailRequest_Tool{Tool: &v1.EventReference{SessionId: item.Event.Reference.Session, Sequence: item.Event.Reference.Sequence}}})
			if err != nil {
				t.Fatal(err)
			}
			found = found || strings.Contains(string(page.Data), "write_file")
		}
	}
	if !found {
		t.Fatal("tool original was not addressable")
	}
	again, err := f.app.Start(ctx, state.Session.ID, "request-approve", "write a file")
	if err != nil || again.ID != task.ID || again.Status != "answered" || f.executed.Load() != 1 {
		t.Fatal(again, err)
	}
}

func TestShutdownCancelsAndJoinsModel(t *testing.T) {
	ctx := deadline(t)
	started, joined := make(chan struct{}), make(chan struct{})
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
		close(started)
		<-ctx.Done()
		close(joined)
		return ctx.Err()
	}), false)
	state, err := f.app.Select(ctx, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.Start(ctx, state.Session.ID, "request-shutdown", "wait"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := f.app.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-joined:
	default:
		t.Fatal("shutdown returned before generation joined")
	}
	if _, err := f.app.Select(ctx, "", true); err == nil {
		t.Fatal("shutdown admitted work")
	}
}
