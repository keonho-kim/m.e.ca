// Purpose: Verify per-call usage snapshots become one persisted record, including interrupted calls.
// Pattern: Runtime behavior tests.
// Usage: go test ./internal/agent/run -run TestTokenUsage.
// Related: internal/agent/run/usage.go, internal/agent/run/generate.go.
package run

import (
	"context"
	"errors"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"testing"
)

type usageGenerator struct{ interrupt bool }

func (g usageGenerator) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	for _, e := range []modelapi.Event{{Delta: "DONE\nhi", Streaming: true}, {Usage: &modelapi.TokenUsage{Input: 20, Output: 2}}, {Usage: &modelapi.TokenUsage{Input: 20, Output: 4}}} {
		if err := emit(e); err != nil {
			return err
		}
	}
	if g.interrupt {
		return context.Canceled
	}
	return emit(modelapi.Event{Done: true, Usage: &modelapi.TokenUsage{Input: 20, Output: 4}})
}
func TestTokenUsageRecordedOnce(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		j := &journal{}
		r := fixture(&generator{}, &executor{}, j)
		r.Generator = usageGenerator{interrupt: interrupted}
		finals := 0
		var first *modelapi.TokenUsage
		_, err := r.Run(context.Background(), "hello", false, approve, func(e Event) error {
			if e.Kind == "usage_progress" && first == nil {
				first = e.Usage
			}
			if e.Kind == "usage" {
				finals++
				if e.Usage == nil || e.Usage.Input != 20 || e.Usage.Output != 4 {
					t.Fatalf("usage=%+v", e.Usage)
				}
			}
			return nil
		})
		if interrupted && !errors.Is(err, context.Canceled) || !interrupted && err != nil {
			t.Fatal(err)
		}
		recorded := 0
		for _, e := range j.events {
			if e.Kind == "usage" {
				recorded++
			}
		}
		if finals != 1 || recorded != 1 {
			t.Fatalf("finals=%d records=%d", finals, recorded)
		}
		if first == nil || first.Output != 2 {
			t.Fatal("queued usage snapshot mutated")
		}
	}
}

func TestTokenUsageAbsentDoesNotEmitOrPersistCounts(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		j := &journal{}
		r := fixture(&generator{}, &executor{}, j)
		g := &chunkGenerator{chunks: []string{"DONE\nhello"}}
		if interrupted {
			g.cancel = cancel
		}
		r.Generator = g
		_, err := r.Run(ctx, "hello", false, approve, func(e Event) error {
			if e.Usage != nil {
				t.Errorf("usage emitted without provider counts: %+v", e.Usage)
			}
			return nil
		})
		cancel()
		if !interrupted && err != nil {
			t.Fatal(err)
		}
		for _, e := range j.events {
			if e.Kind == "usage" {
				t.Fatal("unreported counts persisted")
			}
		}
	}
}
