// Purpose: Verify LAT history reuses its native audit while hiding prompt envelopes and machine verdicts.
// Pattern: Persistence-to-presentation contract regression.
// Usage: go test ./internal/storage/sqlite -run TestLATTimeline.
// Related: internal/storage/sqlite/timeline.go, internal/tui/transcript/lat_conversation.go.
package sqlite

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	lattranscript "github.com/keonho-kim/meca/internal/agent/lat/transcript"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/transcript"
)

func TestLATTimelineRestoresNativeRepliesAndToolsWithoutPromptCopies(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	plan := "VISIBLE_PLAN\n" + strings.Repeat("implementation step\n", 2000)
	for _, phase := range []string{"plan", "review"} {
		j, err := lattranscript.New(s, id, "main_run", phase)
		if err != nil {
			t.Fatal(err)
		}
		call := toolapi.Call{ID: "call_" + phase, Name: "execute_command", Arguments: json.RawMessage(`{"command":"cat README.md","workdir":"."}`)}
		request := modelapi.Message{Role: "user", Content: "HIDDEN_CONTEXT_PACKET"}
		assistant := modelapi.Message{Role: "assistant", Content: "Reading sources", Calls: []toolapi.Call{call}}
		result := modelapi.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Name, Content: `{"stdout":"SOURCE_CONTENT"}`}
		final := modelapi.Message{Role: "assistant", Content: plan}
		if phase == "review" {
			final.Content = "NO_FINDINGS"
		}
		if err = j.Commit(ctx, id, []runrecord.Entry{
			{Message: &request},
			{Event: &runrecord.Event{Kind: "thinking", Thinking: "VISIBLE_THINKING"}},
			{Message: &assistant},
			{Message: &result},
			{Event: &runrecord.Event{Kind: "tool_succeeded", Call: &call, CallID: call.ID, Output: json.RawMessage(`{"Stdout":"SOURCE_CONTENT","ExitCode":0,"Status":"exited"}`)}},
			{Message: &final},
		}); err != nil {
			t.Fatal(err)
		}
		j.Close()
	}
	if err := s.RecordEvent(ctx, id, runrecord.Event{Kind: "lat_clean", RunID: "main_run"}); err != nil {
		t.Fatal(err)
	}
	records, _, err := s.Timeline(ctx, id, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	f := transcript.Formatter{Language: locale.English, Redact: func(s string) string { return s }}
	var shown strings.Builder
	for _, block := range f.History(records) {
		if block.Group.Title != "Look At This" {
			t.Fatalf("LAT history lost group: %+v", block)
		}
		shown.WriteString(block.Text + "\n")
	}
	for _, text := range []string{plan, "VISIBLE_THINKING", "cat README.md", "SOURCE_CONTENT"} {
		if !strings.Contains(shown.String(), text) {
			t.Errorf("restored LAT hid %q", text[:min(50, len(text))])
		}
	}
	for _, text := range []string{"HIDDEN_CONTEXT_PACKET", "NO_FINDINGS"} {
		if strings.Contains(shown.String(), text) {
			t.Errorf("private protocol content shown: %s", text)
		}
	}
	if strings.Count(shown.String(), "VISIBLE_PLAN") != 1 {
		t.Fatal("plan body duplicated in timeline")
	}
}
