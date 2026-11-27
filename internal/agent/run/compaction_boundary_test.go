// Purpose: Verify compaction boundaries admit fresh instructions and bound only consecutive capacity failures.
// Pattern: Scripted model and repository integration tests.
// Usage: go test ./internal/agent/run -run 'TestCompactionBoundary|TestCapacityRecovery'.
// Related: internal/agent/run/run.go, internal/agent/run/memory.go, internal/agent/run/steering.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/modeltest"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type boundaryModel func(context.Context, modelapi.Request, func(modelapi.Event) error) error

func (f boundaryModel) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	return f(ctx, request, emit)
}

type preparingModel struct {
	boundaryModel
	prepare func(context.Context, modelapi.Request) (modelapi.PreparedGeneration, error)
}

func (g preparingModel) Prepare(ctx context.Context, request modelapi.Request) (modelapi.PreparedGeneration, error) {
	return g.prepare(ctx, request)
}

func TestPreparedRequestRebuildsAfterSteering(t *testing.T) {
	j := &journal{}
	r := fixture(&generator{}, &executor{}, j)
	preparations, generations := 0, 0
	var pending []string
	r.PollSteering = func() []string {
		messages := pending
		pending = nil
		return messages
	}
	r.Generator = preparingModel{
		boundaryModel: func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
			t.Fatal("prepared request was generated through the unprepared path")
			return nil
		},
		prepare: func(_ context.Context, request modelapi.Request) (modelapi.PreparedGeneration, error) {
			preparations++
			if preparations == 1 {
				pending = []string{"Use the updated request"}
			}
			return modelapi.PreparedGeneration{Send: func(_ context.Context, emit func(modelapi.Event) error) error {
				generations++
				last := request.Messages[len(request.Messages)-1]
				if last.Content != "Use the updated request" {
					t.Fatalf("stale prepared request sent: %q", last.Content)
				}
				return emit(modelapi.Event{Delta: "done", Done: true})
			}}, nil
		},
	}
	_, err := r.Run(t.Context(), "Original request", approve, emit)
	if err != nil || preparations != 2 || generations != 1 || recordedModelCalls(j) != 1 {
		t.Fatalf("preparations=%d generations=%d calls=%d err=%v", preparations, generations, recordedModelCalls(j), err)
	}
}

func TestCompactionBoundaryAppliesSteeringBeforeNextGeneration(t *testing.T) {
	store := &activityRepository{}
	store.messages = []modelapi.Message{{Role: "user", Content: "old requirement"}}
	r := fixture(&generator{}, &executor{}, &store.journal)
	r.Journal = store
	templates := prompt.Defaults()
	templates.System += "\nSYSTEM_RULE_MARKER"
	r.Prompts = &templates
	r.Memory = &compact.Controller{Repository: store, Session: r.Session, Model: r.Model, Prompt: prompt.Defaults().Compaction, Settings: compact.Defaults(), Redact: r.Redact}
	pending, calls := false, 0
	r.PollSteering = func() []string {
		if pending {
			pending = false
			return []string{"Use the new requirement"}
		}
		return nil
	}
	r.Generator = boundaryModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		if len(request.Tools) == 0 {
			if request.Messages[0].Content != templates.Compaction {
				t.Error("summary generation received main-agent instructions")
			}
			pending = true
			return emit(modelapi.Event{Delta: "- Previous work summarized.", Done: true})
		}
		calls++
		found := false
		for _, m := range request.Messages {
			found = found || m.Role == "user" && m.Content == "Use the new requirement"
		}
		if !found {
			t.Error("first generation after compaction missed the new instruction")
		}
		for _, rule := range []string{r.ProjectInstructions, "SYSTEM_RULE_MARKER"} {
			if strings.Count(request.Messages[0].Content, rule) != 1 {
				t.Errorf("compaction lost or duplicated instruction: %s", rule)
			}
		}
		return emit(modelapi.Event{Delta: "done", Done: true})
	})
	result, err := r.Run(context.Background(), "Original request", approve, emit)
	if err != nil || result.Status != runrecord.Answered || calls != 1 {
		t.Fatalf("calls=%d status=%s error=%v", calls, result.Status, err)
	}
	for _, event := range store.events {
		if g := event.Generation; g != nil && g.BeforeCompaction != nil {
			if g.BeforeCompaction.Input != 245760 || g.BeforeCompaction.Source != "watermark" || g.Context.Source != "estimate" {
				t.Fatalf("before=%+v after=%+v", g.BeforeCompaction, g.Context)
			}
			return
		}
	}
	t.Fatal("missing persisted compaction measurement")
}

func TestCapacityRecoveryResetsAfterSuccessfulResponse(t *testing.T) {
	for _, consecutive := range []bool{false, true} {
		store := &capacityStore{}
		tools := &executor{}
		r := fixture(&generator{}, tools, &store.journal)
		r.Journal = store
		r.Memory = &compact.Controller{Repository: store, Session: r.Session, Model: r.Model, Prompt: prompt.Defaults().Compaction, Settings: compact.Defaults(), Redact: r.Redact}
		calls, summaries := 0, 0
		r.Generator = boundaryModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
			if len(request.Tools) == 0 {
				summaries++
				return emit(modelapi.Event{Delta: "- Continue current work.", Done: true})
			}
			calls++
			if consecutive || calls == 1 || calls == 3 {
				return &modelapi.ContextLimitError{}
			}
			if calls == 2 {
				return emit(modelapi.Event{Done: true, Calls: modeltest.Calls([]toolapi.Call{{ID: "once", Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"new"}`)}})})
			}
			return emit(modelapi.Event{Delta: "done", Done: true})
		})
		result, err := r.Run(context.Background(), "Write a", approve, emit)
		if consecutive {
			if err == nil || calls != 2 || summaries != 1 || tools.executed != 0 {
				t.Fatalf("consecutive failure retried indefinitely: calls=%d summaries=%d executed=%d error=%v", calls, summaries, tools.executed, err)
			}
		} else if err != nil || result.Status != runrecord.Answered || calls != 4 || summaries != 2 || tools.executed != 1 {
			t.Fatalf("later capacity failure not recovered: calls=%d summaries=%d executed=%d error=%v", calls, summaries, tools.executed, err)
		}
	}
}
