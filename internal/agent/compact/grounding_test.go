// Purpose: Verify archived evidence is available to summarization without being appended to memory.
// Pattern: Compaction source and context contract test.
// Usage: go test ./internal/agent/compact -run TestGrounding.
// Related: internal/agent/compact/controller.go, internal/agent/compact/context.go.
package compact

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestGroundingIsSourceOnly(t *testing.T) {
	record, err := grounding.Bind(tool.Call{ID: "c", Name: "read_file", Arguments: []byte(`{"path":"README.md"}`)}, "README.md is unknown.", "run_1")
	if err != nil {
		t.Fatal(err)
	}
	repo := &repository{mark: Watermark{Tokens: 256000, Usage: 1, Timeline: 1}, records: []Record{{Sequence: 1, Grounding: &record}}}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	_, err = c.Refresh(context.Background(), func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		if !strings.Contains(request.Messages[1].Content, repoSourceMarker(repo)) {
			t.Fatal("execution evidence missing from summary source")
		}
		return emit(modelapi.Event{Delta: summary(), Done: true})
	})
	if err != nil {
		t.Fatal(err)
	}
	text, _, err := c.Context()
	if err != nil || text != summary() {
		t.Fatalf("unexpected grounding appendix: %s %v", text, err)
	}
}

func TestFailedDiagnosticIsSourceOnly(t *testing.T) {
	repo := &repository{mark: Watermark{Tokens: 256000, Usage: 1, Timeline: 1}, records: []Record{{Sequence: 1, Operation: &tool.Operation{CallID: "c", RunID: "task", Name: "execute_command", Status: "tool_failed", Process: &tool.ProcessOutcome{Status: "exited", ExitCode: 1, Diagnostic: "src/a:4 expected ready, got missing"}}}}}
	c := Controller{Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	if _, err := c.Refresh(context.Background(), func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		if !strings.Contains(request.Messages[1].Content, repoSourceMarker(repo)) {
			t.Fatal("execution evidence missing from summary source")
		}
		return emit(modelapi.Event{Delta: summary(), Done: true})
	}); err != nil {
		t.Fatal(err)
	}
	text, _, err := c.Context()
	if err != nil || text != summary() {
		t.Fatalf("unexpected diagnostic appendix: %s %v", text, err)
	}
}

func repoSourceMarker(repo *repository) string {
	if repo.records[0].Grounding != nil {
		return repo.records[0].Grounding.Reason
	}
	return repo.records[0].Operation.Process.Diagnostic
}
