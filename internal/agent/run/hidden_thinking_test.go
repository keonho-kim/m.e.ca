// Purpose: Preserve private reasoning and actual usage while avoiding hidden display work.
// Pattern: Scripted normal and interrupted generation regression tests.
// Usage: Team configures HideThinking; ordinary agent streaming keeps its default behavior.
// Related: internal/agent/run/generate.go, internal/agent/run/usage.go.
package run

import (
	"context"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/config/credentials"
)

func TestHiddenThinkingKeepsSanitizedTraceUsageAndCancellation(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		t.Run(map[bool]string{false: "completed", true: "interrupted"}[interrupted], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			j := &journal{}
			r := fixture(&generator{}, &executor{}, j)
			vault := &credentials.Vault{}
			vault.Remember("provider-secret")
			r.Redact, r.NewStreamMask = vault.Redact, vault.NewStreamMask
			r.HideThinking = true
			r.Generator = boundaryModel(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
				for _, text := range []string{"Private provider-", "secret reasoning"} {
					if err := emit(modelapi.Event{Thinking: text, Streaming: true}); err != nil {
						return err
					}
				}
				if err := emit(modelapi.Event{Usage: &modelapi.TokenUsage{Input: 10, Output: 20}}); err != nil {
					return err
				}
				if interrupted {
					cancel()
					return ctx.Err()
				}
				return emit(modelapi.Event{Delta: "Done", Streaming: true, Done: true})
			})
			_, err := r.Run(ctx, "Inspect the service", approve, func(event runrecord.Event) error {
				if event.Kind == "thinking" || event.Thinking != "" {
					t.Error("hidden thinking reached presentation")
				}
				return nil
			})
			if (err != nil) != interrupted {
				t.Fatal("unexpected completion state", err)
			}
			trace, usage := false, false
			for _, event := range j.events {
				if event.Kind == "thinking" {
					trace = strings.Contains(event.Thinking, "[redacted]") && !strings.Contains(event.Thinking, "provider-secret")
				}
				if event.Kind == "usage" && event.Usage != nil {
					usage = event.Usage.Output == 20
				}
			}
			if !trace || !usage {
				t.Fatal("private trace or reported reasoning-inclusive output was lost", trace, usage)
			}
		})
	}
}
