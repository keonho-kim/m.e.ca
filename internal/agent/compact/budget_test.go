// Purpose: Verify summary selection respects context/output budgets without discarding required instructions or prior memory.
// Pattern: Model request boundary tests.
// Usage: go test ./internal/agent/compact -run TestSummaryBudget.
// Related: internal/agent/compact/controller.go, internal/agent/compact/summarize.go.
package compact

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

func TestSummaryBudgetReservesOutputAndFixedInput(t *testing.T) {
	for _, window := range []int{16384, 32768} {
		repo := &repository{memory: Memory{Revision: 1, Through: 1, Markdown: "Keep the agreed API."}, mark: Watermark{Timeline: 102, Usage: 10, Tokens: 250000}}
		repo.records = append(repo.records, Record{Sequence: 2, Role: "user", Text: "Preserve all user constraints."})
		for i := 3; i <= 102; i++ {
			repo.records = append(repo.records, Record{Sequence: int64(i), Role: "observation", Text: fmt.Sprintf("Observation %d: %s", i, strings.Repeat("x", 2000))})
		}
		c := Controller{ContextWindow: window, Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), ActiveRequest: "Finish the current request", Redact: func(s string) string { return s }}
		calls := 0
		changed, err := c.Refresh(context.Background(), nil, func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
			calls++
			size := 0
			for _, message := range request.Messages {
				size += len(message.Content)
			}
			if EstimateBytes(int64(size)).Amount+int64(request.MaxOutput) > int64(window) {
				t.Errorf("summary exceeds %d-token context: input bytes=%d output=%d", window, size, request.MaxOutput)
			}
			for _, required := range []string{"Keep the agreed API", "Preserve all user constraints", "Finish the current request"} {
				if !strings.Contains(request.Messages[1].Content, required) {
					t.Errorf("missing %s", required)
				}
			}
			return emit(modelapi.Event{Delta: "- Updated work state.", Done: true})
		})
		if err != nil || !changed || calls != 1 {
			t.Fatalf("changed=%v calls=%d error=%v", changed, calls, err)
		}
	}
}

func TestSummaryBudgetPreservesUnfittableRequiredInformation(t *testing.T) {
	for _, largePrevious := range []bool{false, true} {
		prior := Memory{Revision: 1, Through: 1, Markdown: "Prior memory"}
		instruction := "Keep all constraints"
		if largePrevious {
			prior.Markdown = strings.Repeat("previous decision ", 4000)
		} else {
			instruction = strings.Repeat("user requirement ", 4000)
		}
		repo := &repository{memory: prior, mark: Watermark{Timeline: 2, Usage: 10, Tokens: 250000}, records: []Record{{Sequence: 2, Role: "user", Text: instruction}}}
		settings := inference.Defaults()
		settings.Compaction.MaxOutputTokens = 1024
		c := Controller{ContextWindow: 8192, Inference: &settings, Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
		calls := 0
		_, err := c.Refresh(context.Background(), nil, func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
			calls++
			return emit(modelapi.Event{Delta: "short summary", Done: true})
		})
		if err == nil || calls != 0 || repo.installed != 0 || repo.memory != prior {
			t.Fatalf("required information discarded: previous=%v calls=%d error=%v", largePrevious, calls, err)
		}
	}
}

func TestSummaryBudgetKeepsLargeFailureExcerpt(t *testing.T) {
	repo := &repository{mark: Watermark{Timeline: 2, Usage: 1, Tokens: 250000}, records: []Record{
		{Sequence: 1, Role: "user", Text: "Fix the failed build"},
		{Sequence: 2, Role: "failure", Text: "main.go:42 failure\n" + strings.Repeat("<&> log\n", 30000) + "\nFinal cause: missing import"},
	}}
	c := Controller{ContextWindow: 16384, Repository: repo, Session: "s", Model: "m", Prompt: prompt.Defaults().Compaction, Settings: Defaults(), Redact: func(s string) string { return s }}
	changed, err := c.Refresh(context.Background(), nil, func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		for _, detail := range []string{"main.go:42 failure", "Final cause: missing import"} {
			if !strings.Contains(request.Messages[1].Content, detail) {
				t.Errorf("large failure lost its useful excerpt: %s", detail)
			}
		}
		return emit(modelapi.Event{Delta: "- Fix the missing import.", Done: true})
	})
	if err != nil || !changed {
		t.Fatalf("changed=%v error=%v", changed, err)
	}
}
