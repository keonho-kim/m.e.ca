// Purpose: Verify PFC and Thinking diagnostics retain the diagnostic role's parameters.
// Pattern: Generator contract test.
// Usage: go test ./internal/provider/diagnostic -run TestInferenceDiagnostics.
// Related: internal/provider/diagnostic/check.go, internal/provider/diagnostic/effort.go.
package diagnostic

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"testing"
)

type generatorFunc func(context.Context, modelapi.Request, func(modelapi.Event) error) error

func (f generatorFunc) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	return f(ctx, r, emit)
}
func TestInferenceDiagnostics(t *testing.T) {
	roles := inference.Defaults()
	roles.Diagnostic.Temperature = 0.05
	roles.Diagnostic.MaxOutputTokens = 1024
	calls := 0
	generate := generatorFunc(func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		if r.Temperature == nil || *r.Temperature != 0.05 || r.MaxOutput != 1024 {
			t.Fatalf("diagnostic settings lost: %+v", r)
		}
		text := "391"
		question := r.Messages[len(r.Messages)-1].Content
		if strings.Contains(question, "Reply exactly TOOL") {
			text = "TOOL read_file"
		} else if strings.Contains(question, "ARGUMENT") {
			text = "AGENTS.md"
		}
		thinking := ""
		if r.Thinking != nil && *r.Thinking {
			thinking = "Checking the product."
		}
		return emit(modelapi.Event{Delta: text, Thinking: thinking, Done: true})
	})
	caps, err := Check(context.Background(), generate, "alias", true, true, &roles)
	if err != nil || !caps.Thinking || !caps.ToolDialogue {
		t.Fatalf("caps=%+v %v", caps, err)
	}
	if err := Effort(context.Background(), generate, "alias", "high", &roles); err != nil {
		t.Fatal(err)
	}
	if calls != 5 {
		t.Fatalf("calls=%d", calls)
	}
}
