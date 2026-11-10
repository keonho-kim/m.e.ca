// Purpose: Rebuild an in-flight streamed answer from bounded snapshot previews and revision-bound pages.
// Pattern: Controlled native stream recovery integration test.
// Usage: go test -race ./internal/serverapi -run TestDraftPages.
// Related: internal/commands/server/draft_original.go, api/meca/v1/workspace.proto.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/config/credentials"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"strings"
	"testing"
)

func TestDraftPagesRecoverBeyondSnapshotWithoutStoppingTask(t *testing.T) {
	text := strings.Repeat("가나다", 40000)
	release := make(chan struct{})
	emitted := make(chan struct{})
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
		if err := emit(model.Event{Delta: text, Streaming: true}); err != nil {
			return err
		}
		close(emitted)
		select {
		case <-release:
			return emit(model.Event{Done: true})
		case <-ctx.Done():
			return ctx.Err()
		}
	}), false)
	transport := nativeTransport(t, f, "sse")
	started, cursor := startNativeTask(t, f, "paged-live-draft", "stream")
	<-emitted
	state := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_State{State: &v1.StateRequest{TaskId: started.ID}}}).GetState()
	if len(state.GetDrafts()) != 1 || !state.GetDrafts()[0].GetTruncated() || len(state.GetDrafts()[0].GetText()) > 4096 {
		t.Fatal("snapshot retained an unbounded draft", state)
	}
	draft := state.GetDrafts()[0]
	var result strings.Builder
	var revision *int64
	for int64(result.Len()) < draft.GetTextBytes() {
		reply := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Draft{Draft: &v1.DraftRequest{TaskId: started.ID, Agent: draft.GetAgent(), Generation: draft.GetGeneration(), Offset: int64(result.Len()), Limit: 64 << 10, Revision: revision}}})
		page := reply.GetDraft()
		if page == nil || page.GetNextOffset() <= int64(result.Len()) {
			t.Fatal(reply)
		}
		revision = new(page.GetRevision())
		result.WriteString(page.GetText())
	}
	if result.String() != text {
		t.Fatal("paged source lost streamed bytes")
	}
	wrong := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Draft{Draft: &v1.DraftRequest{TaskId: started.ID, Generation: draft.GetGeneration() + 1, Limit: 64 << 10, Revision: revision}}})
	if wrong.GetError().GetCode() != "conflict" {
		t.Fatal("stale generation accepted", wrong)
	}
	close(release)
	waitNativeTask(t, deadline(t), f.app, started.ID, cursor)
	page := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_ListTasks{ListTasks: &v1.TaskListRequest{SessionId: started.Session, Limit: 1}}}).GetTasks()
	if len(page.GetItems()) != 1 || page.GetItems()[0].GetStatus() != "answered" {
		t.Fatal(page)
	}
}

func TestDraftPagesMaskCredentialsAcrossPageBoundaries(t *testing.T) {
	text := strings.Repeat("x", (64<<10)-4) + "private-test-key" + "tail"
	emitted := make(chan struct{})
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
		if err := emit(model.Event{Delta: text, Streaming: true}); err != nil {
			return err
		}
		close(emitted)
		<-ctx.Done()
		return ctx.Err()
	}), false)
	vault := &credentials.Vault{}
	if err := vault.Remember("private-test-key"); err != nil {
		t.Fatal(err)
	}
	f.api.RedactRange, f.api.RedactionVersion = vault.RedactRange, vault.RedactionVersion
	transport := nativeTransport(t, f, "sse")
	task, cursor := startNativeTask(t, f, "masked-page-boundary", "hold")
	<-emitted
	state := f.app.State()
	draft := state.Drafts[0]
	request := &v1.DraftRequest{TaskId: task.ID, Agent: draft.Agent, Generation: draft.Generation, Limit: 64 << 10}
	first := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Draft{Draft: request}}).GetDraft()
	if first == nil || strings.Contains(first.GetText(), "priv") || !strings.Contains(first.GetText(), "[redacted]") {
		t.Fatal("page exposed a credential prefix")
	}
	request.Offset, request.Revision = first.GetNextOffset(), new(first.GetRevision())
	second := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Draft{Draft: request}}).GetDraft()
	if second == nil || strings.Contains(second.GetText(), "ate-test-key") {
		t.Fatal("page exposed a credential suffix")
	}
	if err := vault.Remember("another-key"); err != nil {
		t.Fatal(err)
	}
	stale := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_Draft{Draft: request}})
	if stale.GetError().GetCode() != "conflict" {
		t.Fatal(stale)
	}
	if err := f.app.Cancel(task.ID); err != nil {
		t.Fatal(err)
	}
	waitNativeTask(t, deadline(t), f.app, task.ID, cursor)
}
