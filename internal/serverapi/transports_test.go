// Purpose: Verify streamed-before-final output, approvals, replay and identical tool effects across transports.
// Pattern: Shared observable protocol contract tests.
// Usage: go test ./internal/serverapi -run TestTransport.
// Related: internal/serverapi/wire_test.go, internal/serverapi/fixture_test.go.
package serverapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestTransportStreamingApprovalAndReconnect(t *testing.T) {
	var baseline []string
	for _, kind := range []string{"sse", "socket", "grpc", "grpc-web", "connect"} {
		t.Run(kind, func(t *testing.T) {
			ctx := deadline(t)
			release := make(chan struct{})
			var calls atomic.Int32
			f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
				if calls.Add(1) > 1 {
					return emit(model.Event{Delta: "complete", Done: true})
				}
				if err := emit(model.Event{Delta: strings.Repeat("before tool ", 20), Streaming: true}); err != nil {
					return err
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-release:
				}
				return emit(model.Event{Calls: []tool.Call{{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"path":"transport.txt","content":"same result"}`)}}, Done: true})
			}), true)
			w := tlsWire(t, kind, f.api)
			state := call(t, ctx, w, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
			watch, err := w.watch(ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			initial := next(t, watch)
			if initial.GetSnapshot() == nil || initial.GetSnapshot().Session.Id != state.Session.Id {
				t.Fatal(initial)
			}
			started := call(t, ctx, w, &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: state.Session.Id, Key: "request-once", Text: "write file"}}}).GetTask()
			cursor := ""
			for cursor == "" {
				n := next(t, watch)
				if n.GetEvent().GetKind() == "assistant_delta" && strings.HasPrefix(n.GetEvent().GetMessage().GetText(), "before tool") {
					cursor = n.GetEvent().Cursor
				}
			}
			if f.app.State().Task.Status != "running" {
				t.Fatal("stream arrived only after completion")
			}
			if err := watch.close(); err != nil {
				t.Fatal(err)
			}
			again := call(t, ctx, w, &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: state.Session.Id, Key: "request-once", Text: "write file"}}}).GetTask()
			if again.Id != started.Id {
				t.Fatal("retry changed task identity")
			}
			close(release)
			watch, err = w.watch(ctx, cursor)
			if err != nil {
				t.Fatal(err)
			}
			defer watch.close()
			approval := ""
			for approval == "" {
				n := next(t, watch)
				if n.GetSnapshot() != nil {
					for _, p := range n.GetSnapshot().Approvals {
						approval = p.Id
					}
				}
				if n.GetEvent().GetKind() == "approval_required" {
					approval = n.GetEvent().ApprovalId
				}
			}
			call(t, ctx, w, &v1.Command{Operation: &v1.Command_Approve{Approve: &v1.ApprovalDecision{TaskId: started.Id, ApprovalId: approval, Allow: true}}})
			for {
				n := next(t, watch)
				if n.GetEvent().GetKind() == "request_finished" {
					break
				}
			}
			status := call(t, ctx, w, &v1.Command{Operation: &v1.Command_TaskStatus{TaskStatus: &v1.TaskRequest{TaskId: started.Id}}}).GetTask()
			if status.Status != "answered" || calls.Load() != 2 || f.executed.Load() != 1 {
				t.Fatal(status, calls.Load(), f.executed.Load())
			}
			body, err := os.ReadFile(filepath.Join(f.project, "transport.txt"))
			if err != nil || string(body) != "same result" {
				t.Fatal(string(body), err)
			}
			page := call(t, ctx, w, &v1.Command{Operation: &v1.Command_History{History: &v1.HistoryRequest{Limit: 100}}}).GetHistory()
			if len(page.Items) < 3 {
				t.Fatal("missing durable history", page)
			}
			journal := journalSignature(t, ctx, f, state.Session.Id)
			if baseline == nil {
				baseline = journal
			} else if !reflect.DeepEqual(baseline, journal) {
				t.Fatal("transport changed ordered durable execution", journal, baseline)
			}
			stale, err := w.call(ctx, &v1.Command{Operation: &v1.Command_Approve{Approve: &v1.ApprovalDecision{TaskId: started.Id, ApprovalId: approval, Allow: true}}})
			if err != nil || stale.GetError().GetCode() != "conflict" {
				t.Fatal(stale, err)
			}
		})
	}
}

func journalSignature(t *testing.T, ctx context.Context, f *fixture, session string) []string {
	t.Helper()
	var original bytes.Buffer
	if err := f.store.Export(ctx, session, &original); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&original)
	var signature []string
	for {
		var row struct {
			Kind string
			Body json.RawMessage
		}
		if err := decoder.Decode(&row); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Kind, Role, Content string
			Call                *tool.Call
		}
		if err := json.Unmarshal(row.Body, &body); err != nil {
			t.Fatal(err)
		}
		part := row.Kind + ":" + body.Kind + ":" + body.Role
		if body.Role == "assistant" || body.Role == "user" {
			part += ":" + body.Content
		}
		if body.Call != nil {
			part += ":" + body.Call.Name + ":" + string(body.Call.Arguments)
		}
		signature = append(signature, part)
	}
	return signature
}

func TestTransportCancel(t *testing.T) {
	for _, kind := range []string{"sse", "socket", "grpc"} {
		t.Run(kind, func(t *testing.T) {
			ctx := deadline(t)
			started := make(chan struct{})
			f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
				close(started)
				<-ctx.Done()
				return ctx.Err()
			}), false)
			w := tlsWire(t, kind, f.api)
			state := call(t, ctx, w, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
			task := call(t, ctx, w, &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: state.Session.Id, Key: "request-cancel", Text: "wait"}}}).GetTask()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			call(t, ctx, w, &v1.Command{Operation: &v1.Command_Cancel{Cancel: &v1.TaskRequest{TaskId: task.Id}}})
			finished := waitFinished(t, ctx, f.app, "")
			if finished.Status != "interrupted" {
				t.Fatal(finished)
			}
		})
	}
}
