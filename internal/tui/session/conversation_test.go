// Purpose: Verify conversational turn ordering and the editor below transcript output.
// Pattern: Terminal behavior tests.
// Usage: go test ./internal/tui/session -run 'TestConversation|TestTranscript'.
// Related: internal/tui/session/render.go, internal/tui/session/history.go, internal/tui/session/events.go.
package session

import (
	"encoding/json"
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/run"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
	"strings"
	"testing"
)

func TestConversationEditorFollowsTurns(t *testing.T) {
	a := fixture(t)
	for _, text := range []string{"first question", "second question"} {
		for _, block := range (transcript.Formatter{Redact: a.opts.Vault.Redact}).Message(modelapi.Message{Role: "user", Content: text}) {
			a.output.Append(block.Text, block.Thinking)
		}
		a.receive(notice{Event: &run.Event{Kind: "response", Text: "answer to " + text}})
		a.receive(notice{Done: true})
	}
	a.selection.Model = "chosen"
	a.editor.Set("next draft")
	a.render()
	var rows []string
	for y := 0; y < 24; y++ {
		rows = append(rows, strings.TrimSpace(screenLine(a.screen, y, 80)))
	}
	want := []string{"INPUT 0 OUTPUT 0", "> first question", "", "Agent", "answer to first question", "", "", "> second question", "", "Agent", "answer to second question", "", "", "> next draft"}
	for y, text := range want {
		if rows[y] != text {
			t.Fatalf("row %d = %q; want %q\n%s", y, rows[y], text, strings.Join(rows, "\n"))
		}
	}
	if strings.Contains(strings.Join(rows, "\n"), "Model answer") {
		t.Fatal("completion boilerplate displayed")
	}
}

func TestTranscriptAnsweredCheckpointHidden(t *testing.T) {
	a := fixture(t)
	event := run.Event{Checkpoint: &run.Checkpoint{Status: run.Answered, Phase: "finished"}}
	if blocks := (transcript.Formatter{Redact: a.opts.Vault.Redact}).Event(event); len(blocks) != 0 {
		t.Fatalf("completion boilerplate restored: %+v", blocks)
	}
}

func TestConversationHintsScrollAndResize(t *testing.T) {
	a := fixture(t)
	a.output.Append(strings.Repeat("history\n", 40)+"latest answer", false)
	typeInput(a, "!")
	a.screen.SetSize(32, 12)
	a.render()
	inputRow := -1
	for y := 0; y < 12; y++ {
		if strings.TrimSpace(screenLine(a.screen, y, 32)) == "> !" {
			inputRow = y
		}
	}
	if inputRow < 2 || !strings.Contains(screenLine(a.screen, inputRow+1, 32), "!plan") {
		t.Fatal("editor and candidates not below history")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	a.editor.Set("line one\nline two")
	a.render()
	if strings.TrimSpace(screenLine(a.screen, 10, 32)) != "line two" {
		t.Fatal("multiline editor not at conversation end")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyPgUp, "", 0))
	if a.output.Follow {
		t.Fatal("PageUp did not leave follow mode")
	}
	id, row := a.output.AnchorID, a.output.AnchorRow
	a.receive(notice{Event: &run.Event{Kind: "stream", Text: "streamed answer", Thinking: "inspect"}})
	a.screen.SetSize(40, 16)
	a.render()
	if a.output.Follow || a.output.AnchorID != id || a.output.AnchorRow != row || a.editor.String() != "line one\nline two" {
		t.Fatal("stream or resize reset reading position/draft")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyPgDn, "", 0))
	a.render()
	if !a.output.Follow {
		t.Fatal("PageDown did not return to latest output")
	}
}

func TestConversationResumesToolTurns(t *testing.T) {
	a := fixture(t)
	if err := a.opts.Store.AppendMessage(a.ctx, a.opts.Session.ID, modelapi.Message{Role: "user", Content: "inspect file"}); err != nil {
		t.Fatal(err)
	}
	if err := a.opts.Store.RecordEvent(a.ctx, a.opts.Session.ID, run.Event{Kind: "tool_succeeded", CallID: "call", Call: &toolapi.Call{ID: "call", Name: "read_file", Arguments: json.RawMessage(`{"path":"README.md"}`)}, Output: json.RawMessage(`{"content":"file inspected"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := a.opts.Store.AppendMessage(a.ctx, a.opts.Session.ID, modelapi.Message{Role: "assistant", Content: "file inspected"}); err != nil {
		t.Fatal(err)
	}
	if err := a.loadHistory(false); err != nil {
		t.Fatal(err)
	}
	a.render()
	for y, text := range []string{"INPUT 0 OUTPUT 0", "> inspect file", "", "Agent", "", "read_file", "", "path: README.md", "", "Output", "content: file inspected", "", "file inspected", "", "", ">"} {
		if got := strings.TrimSpace(screenLine(a.screen, y, 80)); got != text {
			t.Fatalf("restored row %d=%q; want %q", y, got, text)
		}
	}
}
