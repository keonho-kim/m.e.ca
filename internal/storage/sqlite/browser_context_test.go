// Purpose: Verify browser originals share storage and consumed observations do not hydrate into model context.
// Pattern: SQLite journal contract regression with a bounded restoration budget.
// Usage: go test ./internal/storage/sqlite -run TestBrowserObservation.
// Related: internal/storage/sqlite/observation_context.go, internal/storage/sqlite/memory_sources.go.
package sqlite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestBrowserObservationRetiresBeforeRestoration(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, id := t.Context(), store.Session().ID
	const pages = 128
	const maxContext = 1 << 20
	entries := make([]runrecord.Entry, 0, 2*pages)
	for index := range pages {
		call := toolapi.Call{ID: fmt.Sprintf("call_%d", index), Name: toolapi.BrowserInspect, Arguments: json.RawMessage(`{"view":"text"}`)}
		body := fmt.Sprintf(`{"result_id":"result_%d","text":"%s"}`, index, strings.Repeat("x", (16<<10)-128))
		entries = append(entries,
			runrecord.Entry{Message: &model.Message{Role: "assistant", Calls: []toolapi.Call{call}}},
			runrecord.Entry{Message: &model.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: body, TransientObservation: index%2 == 0}})
	}
	if err := store.Commit(ctx, id, entries); err != nil {
		t.Fatal(err)
	}
	image, err := media.New("viewport.png", "image/png", "png", 128, 128, bytes.Repeat([]byte("pixels"), 256<<10))
	if err != nil {
		t.Fatal(err)
	}
	shot := toolapi.Call{ID: "shot", Name: toolapi.BrowserInspect, Arguments: json.RawMessage(`{"view":"screenshot"}`)}
	if err := store.Commit(ctx, id, []runrecord.Entry{
		{Message: &model.Message{Role: "assistant", Calls: []toolapi.Call{shot}}},
		{Message: &model.Message{Role: "tool", ToolCallID: shot.ID, ToolName: shot.Name, Content: `{"untrusted":true}`, Images: []media.Image{image}, TransientObservation: true}},
		{Message: &model.Message{Role: "assistant", Content: "draft", Incomplete: true}},
	}); err != nil {
		t.Fatal(err)
	}
	const screenshotSequence = 2*pages + 2
	if got, err := store.ContextMessages(ctx, id, 0, maxContext, true, nil); err == nil || len(got) != 0 {
		t.Fatal("an incomplete decision acknowledged unread image")
	}
	boundary, err := store.CompactionBoundary(ctx, id, screenshotSequence+1, 0)
	if err != nil || boundary != screenshotSequence-2 {
		t.Fatal("unread image and its native call were not protected", boundary, err)
	}
	archived, err := store.recordPage(ctx, "SELECT t.sequence,t.kind,m.body,m.refs,m.raw_bytes FROM timeline t JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence WHERE t.sequence=?", screenshotSequence)
	if err != nil || len(archived) != 1 {
		t.Fatal("browser screenshot archive missing", err)
	}
	original, _, err := store.contextRecord(ctx, archived[0], nil, nil)
	if err != nil || len(original.Images) != 1 || original.Images[0].Digest != image.Digest || !bytes.Equal(original.Images[0].Data, image.Data) {
		t.Fatal("browser screenshot archive differs", err)
	}
	if err := store.AppendMessage(ctx, id, model.Message{Role: "assistant", Content: "Inspection finished"}); err != nil {
		t.Fatal(err)
	}
	// Corruption must still be detectable by original readers. The model and
	// compaction projections have no reason to touch these retired bodies.
	if _, err := store.db.ExecContext(ctx, "UPDATE payloads SET body=x'00'"); err != nil {
		t.Fatal(err)
	}
	var charged int
	messages, err := store.ContextMessages(ctx, id, 0, maxContext, true, func(n int) error { charged = n; return nil })
	if err != nil || len(messages) != 2*pages+4 || charged >= 64<<10 {
		t.Fatalf("context messages=%d charged=%d: %v", len(messages), charged, err)
	}
	for _, m := range messages {
		if m.Role == "tool" && (m.Content != toolapi.ObservationRead || len(m.Images) != 0 || m.TransientObservation) {
			t.Fatal("consumed observation retained its body")
		}
	}
	for cursor := int64(0); cursor < screenshotSequence+2; {
		_, next, err := store.MemoryRecords(ctx, id, cursor, screenshotSequence+2)
		if err != nil || next <= cursor {
			t.Fatal("compaction hydrated a retired payload or failed to move", cursor, next, err)
		}
		cursor = next
	}
	if _, err := store.CompactionBoundary(ctx, id, screenshotSequence+2, 32<<10); err != nil {
		t.Fatal("compaction tail read a retired payload", err)
	}
	if _, _, err := store.contextRecord(ctx, archived[0], nil, nil); err == nil {
		t.Fatal("original image lost its integrity check")
	}
	t.Logf("%d completed text pages and one screenshot accounted for %d bytes of decision context", pages, charged)
}

func TestRejectedCallsKeepObservationAndCommitIndividualReceipts(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx, id := t.Context(), store.Session().ID
	inspect := toolapi.Call{ID: "page", Name: toolapi.BrowserInspect, Arguments: json.RawMessage(`{"view":"text"}`)}
	edit := toolapi.Call{ID: "edit", Name: "edit_file", Arguments: json.RawMessage(`{"path":"note.txt"}`)}
	write := toolapi.Call{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"path":"other.txt","reason":"Save the result","content":"text"}`)}
	const evidence = "Exact content needed for the corrected edit"
	var editRef, writeRef runrecord.EventReference
	if err := store.Commit(ctx, id, []runrecord.Entry{
		{Message: &model.Message{Role: "user", Content: "Apply the observed changes."}},
		{Message: &model.Message{Role: "assistant", Calls: []toolapi.Call{inspect}}},
		{Message: &model.Message{Role: "tool", ToolCallID: inspect.ID, ToolName: inspect.Name, Content: evidence, TransientObservation: true}},
		{Message: &model.Message{Role: "assistant", Calls: []toolapi.Call{edit, write}, PreserveObservations: true}},
		{Message: &model.Message{Role: "tool", ToolCallID: edit.ID, ToolName: edit.Name, Content: "missing reason"}},
		{Event: &runrecord.Event{Kind: "tool_failed", CallID: edit.ID, Call: &edit}, Receipt: &editRef},
		{Message: &model.Message{Role: "tool", ToolCallID: write.ID, ToolName: write.Name, Content: "skipped"}},
		{Event: &runrecord.Event{Kind: "tool_failed", CallID: write.ID, Call: &write}, Receipt: &writeRef},
	}); err != nil {
		t.Fatal(err)
	}
	for _, receipt := range []struct {
		ref runrecord.EventReference
		id  string
	}{{editRef, edit.ID}, {writeRef, write.ID}} {
		event, err := store.ToolRequest(ctx, receipt.ref, history.ReadBounds{})
		if err != nil || event.Call == nil || event.Call.ID != receipt.id {
			t.Fatalf("receipt=%+v event=%+v err=%v", receipt, event, err)
		}
	}
	messages, err := store.ContextMessages(ctx, id, 0, 1<<20, true, nil)
	if err != nil || len(messages) != 6 || messages[2].Content != evidence || !messages[2].TransientObservation || !messages[3].PreserveObservations {
		t.Fatalf("restored correction lost its input: %+v %v", messages, err)
	}
	if boundary, err := store.CompactionBoundary(ctx, id, 8, 0); err != nil || boundary != 1 {
		t.Fatalf("unread observation is not protected: boundary=%d err=%v", boundary, err)
	}
	if err := store.AppendMessage(ctx, id, model.Message{Role: "assistant", Content: "Corrected edit complete."}); err != nil {
		t.Fatal(err)
	}
	messages, err = store.ContextMessages(ctx, id, 0, 1<<20, true, nil)
	if err != nil || len(messages) != 7 || messages[2].Content != toolapi.ObservationRead || messages[2].TransientObservation {
		t.Fatalf("validated decision did not release prior input: %+v %v", messages, err)
	}
}

func TestBrowserObservationMessageAndEventShareMediumBody(t *testing.T) {
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, id := t.Context(), store.Session().ID
	result, err := json.Marshal(map[string]string{"result_id": "result_test", "text": strings.Repeat("<á😀>\n", 600)})
	if err != nil {
		t.Fatal(err)
	}
	call := toolapi.Call{ID: "one", Name: toolapi.BrowserInspect, Arguments: json.RawMessage(`{"view":"text"}`)}
	message := model.Message{Role: "tool", ToolName: call.Name, ToolCallID: call.ID, Content: string(result), TransientObservation: true}
	event := runrecord.Event{Kind: "tool_succeeded", CallID: call.ID, Call: &call, Output: result}
	if err := store.Commit(ctx, id, []runrecord.Entry{{Message: &message}, {Event: &event}}); err != nil {
		t.Fatal(err)
	}
	var count, stored int
	if err := store.db.QueryRowContext(ctx, "SELECT count(*),COALESCE(SUM(length(body)),0) FROM payloads").Scan(&count, &stored); err != nil {
		t.Fatal(err)
	}
	if count != 1 || stored != len(result) {
		t.Fatal("browser output does not have a single original", stored, count, len(result))
	}
	restored, err := store.ContextMessages(ctx, id, 0, 64<<10, true, nil)
	if err != nil || len(restored) != 1 || restored[0].Content != string(result) || !restored[0].TransientObservation {
		t.Fatal("unread model result differs", err)
	}
	detail, err := store.ToolEvent(ctx, runrecord.EventReference{Session: id, Sequence: 1}, history.ReadBounds{})
	if err != nil || !bytes.Equal(detail.Output, result) {
		t.Fatal("tool event original differs", err)
	}
}
