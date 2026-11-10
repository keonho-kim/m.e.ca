// Purpose: Verify browser commands, revision-bound editing and submission lookup use the native contract.
// Pattern: Application adapter integration tests.
// Usage: go test ./internal/serverapi -run TestApplication.
// Related: internal/serverapi/application.go, internal/serverapi/skills.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestApplicationCommandsAndEditor(t *testing.T) {
	f := newFixture(t, nil, false)
	transport := nativeTransport(t, f, "sse")
	catalog := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_CommandCatalog{CommandCatalog: &v1.Empty{}}}).GetCommandCatalog()
	if len(catalog.GetItems()) != 6 {
		t.Fatal(catalog)
	}
	for _, text := range []string{"!new", "!quit", ":q"} {
		reply := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_ParseInput{ParseInput: &v1.ParseInput{Text: text}}})
		if reply.GetError().GetCode() != "invalid_argument" {
			t.Fatal(text, reply)
		}
	}
	parsed := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_ParseInput{ParseInput: &v1.ParseInput{Text: `!dir "folder with spaces"`}}}).GetParsedInput()
	if parsed.GetAction() != "dir" || parsed.GetArgument() != "folder with spaces" {
		t.Fatal(parsed)
	}
	path := filepath.Join(f.project, "edit.txt")
	if err := os.WriteFile(path, []byte("initial\n"), 0600); err != nil {
		t.Fatal(err)
	}
	doc := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_EditorRead{EditorRead: &v1.EditFile{Path: "edit.txt"}}}).GetEditor()
	if !doc.GetEditable() || doc.GetText() != "initial\n" || doc.GetRevision() == "" {
		t.Fatal(doc)
	}
	save := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_EditorSave{EditorSave: &v1.EditFile{Path: "edit.txt", Text: "updated\n", Revision: doc.Revision}}})
	if save.GetEditor().GetRevision() == doc.Revision || save.GetEditor().GetText() != "" {
		t.Fatal(save)
	}
	conflict := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_EditorSave{EditorSave: &v1.EditFile{Path: "edit.txt", Text: "stale", Revision: doc.Revision}}})
	if conflict.GetError().GetCode() != "conflict" {
		t.Fatal(conflict)
	}
	wrong := nativeCommand(t, transport, &v1.Command{WorkspaceId: "unknown", Operation: &v1.Command_Capabilities{Capabilities: &v1.Empty{}}})
	if wrong.GetError().GetCode() != "not_found" {
		t.Fatal("workspace silently fell back", wrong)
	}
}

func TestApplicationLookupNeverRepeatsGeneration(t *testing.T) {
	var calls atomic.Int32
	f := newFixture(t, generateFunc(func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
		calls.Add(1)
		return emit(model.Event{Delta: "done", Done: true, Usage: &model.TokenUsage{Input: 7, Output: 5}})
	}), false)
	task, cursor := startNativeTask(t, f, "lookup-original-key", "one request")
	waitNativeTask(t, deadline(t), f.app, task.ID, cursor)
	batch, err := f.app.Replay(cursor)
	if err != nil {
		t.Fatal(err)
	}
	measured := false
	for _, event := range batch.Events {
		if event.Task == task.ID && event.Record.Kind == "request_finished" {
			measured = event.Record.Statistics != nil && event.Record.Statistics.Usage != nil && event.Record.Statistics.Usage.Output == 5
		}
	}
	if !measured {
		t.Fatal("terminal stream lost final usage statistics")
	}
	second, err := f.app.Start(deadline(t), task.Session, "lookup-second-key", "second request")
	if err != nil {
		t.Fatal(err)
	}
	waitNativeTask(t, deadline(t), f.app, second.ID, cursor)
	transport := nativeTransport(t, f, "sse")
	for range 3 {
		reply := nativeCommand(t, transport, &v1.Command{Operation: &v1.Command_LookupRequest{LookupRequest: &v1.RequestLookup{SessionId: task.Session, Key: "lookup-original-key"}}})
		if reply.GetTask().GetId() != task.ID {
			t.Fatal(reply)
		}
		if reply.GetTask().GetOutcome() != "completed" {
			t.Fatal("lost a retained successful outcome", reply)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("lookup repeated generation", calls.Load())
	}
}
