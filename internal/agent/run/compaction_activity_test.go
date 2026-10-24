// Purpose: Verify compaction activity covers source preparation and is closed after failures.
// Pattern: Runtime lifecycle tests with an observable compaction repository.
// Usage: go test ./internal/agent/run -run TestCompactionActivity.
// Related: internal/agent/run/memory.go, internal/agent/compact/controller.go.
package run

import (
	"context"
	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

type activityRepository struct {
	capacityStore
	active, preparedWhileActive, installedWhileActive bool
	preparationError                                  error
}

func (s *activityRepository) MemoryWatermark(ctx context.Context, session string, after int64) (compact.Watermark, error) {
	mark, err := s.capacityStore.MemoryWatermark(ctx, session, after)
	if s.memory.Revision == 0 {
		mark.Tokens = 245760
	}
	return mark, err
}

func (s *activityRepository) MemoryRecords(ctx context.Context, session string, after, through int64) ([]compact.Record, int64, error) {
	s.preparedWhileActive = s.active
	if s.preparationError != nil {
		return nil, after, s.preparationError
	}
	return s.capacityStore.MemoryRecords(ctx, session, after, through)
}

func (s *activityRepository) InstallMemory(ctx context.Context, session string, revision int, memory compact.Memory) error {
	s.installedWhileActive = s.active
	return s.capacityStore.InstallMemory(ctx, session, revision, memory)
}

func TestCompactionActivityCoversPreparationAndFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		failure error
	}{{"success", nil}, {"preparation failure", errors.New("source read failed")}} {
		t.Run(test.name, func(t *testing.T) {
			failure := test.failure
			base := &generator{replies: []string{"Observed the work"}}
			model := &capacityGenerator{base: base}
			store := &activityRepository{preparationError: failure}
			store.messages = []modelapi.Message{{Role: "user", Content: "Earlier work"}}
			r := fixture(base, &executor{}, &store.journal)
			r.Generator, r.Journal = model, store
			r.Memory = &compact.Controller{Repository: store, Session: r.Session, Model: r.Model, Prompt: prompt.Defaults().Compaction, Settings: compact.Defaults(), Redact: r.Redact}
			starts, finishes := 0, 0
			_, err := r.Run(context.Background(), "Continue work", approve, func(event Event) error {
				switch event.Kind {
				case "compaction_started":
					starts++
					store.active = true
				case "compaction_finished":
					finishes++
					store.active = false
				}
				return nil
			})
			if !errors.Is(err, failure) || starts != 1 || finishes != 1 || store.active || !store.preparedWhileActive {
				t.Fatalf("starts=%d finishes=%d active=%t preparedWhileActive=%t err=%v", starts, finishes, store.active, store.preparedWhileActive, err)
			}
			if failure == nil && !store.installedWhileActive || failure != nil && model.summaries != 0 {
				t.Fatal("activity ended before memory commit or generation ran after preparation failed")
			}
		})
	}
}
