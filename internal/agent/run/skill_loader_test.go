// Purpose: Verify explicit instruction loading through the runtime's narrow I/O contract.
// Pattern: Request behavior test with a deterministic non-filesystem loader.
// Usage: go test ./internal/agent/run -run TestSkillLoaderContract.
// Related: internal/agent/run/runner.go, internal/agent/run/skills.go, internal/skills/instruction/document.go.
package run

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/skills/instruction"
)

type instructionLoader struct {
	calls    int
	selected instruction.Selector
	explicit bool
}

func (l *instructionLoader) Load(_ context.Context, selected instruction.Selector, explicit bool) (instruction.Document, error) {
	l.calls++
	l.selected, l.explicit = selected, explicit
	return instruction.Document{Metadata: instruction.Metadata{ID: "project:review", Name: "review", Source: "project"}, Revision: strings.Repeat("a", 64), Body: "instruction-loader-contract-canary"}, nil
}

func TestSkillLoaderContract(t *testing.T) {
	g := &generator{replies: []string{"Done"}}
	r := fixture(g, &executor{}, &journal{})
	loader := &instructionLoader{}
	r.Skills = loader
	if _, err := r.Run(t.Context(), "Use #review", approve, emit); err != nil {
		t.Fatal(err)
	}
	if loader.calls != 1 || loader.selected.ID != "review" || !loader.explicit {
		t.Fatal("explicit read intent did not reach the loader", loader)
	}
	copies := 0
	for _, m := range g.requests[0].Messages {
		copies += strings.Count(m.Content, "instruction-loader-contract-canary")
	}
	if copies != 1 {
		t.Fatal("loaded instructions missing or duplicated", copies)
	}
}
