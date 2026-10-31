// Purpose: Release provider-owned run resources after completion, interruption or journal failure.
// Pattern: Generator lifecycle contract tests.
// Usage: Run TestGeneratorScope with the scripted runner fixture.
// Related: internal/agent/run/run.go, internal/provider/httpclient/run_generator.go.
package run

import (
	"context"
	"fmt"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type scopedGenerator struct {
	base              *generator
	created, released int
	active            bool
}

func (g *scopedGenerator) NewRunGenerator() (modelapi.Generator, func()) {
	g.created++
	g.active = true
	return g, func() { g.active = false; g.released++ }
}

func (g *scopedGenerator) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	if !g.active {
		return fmt.Errorf("generation outside its request lifetime")
	}
	return g.base.Generate(ctx, request, emit)
}

func TestGeneratorScope(t *testing.T) {
	for _, outcome := range []string{"complete", "interrupt", "journal failure"} {
		t.Run(outcome, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			g := &scopedGenerator{base: &generator{replies: []string{"answer", "next answer"}}}
			j := &journal{}
			if outcome == "interrupt" {
				g.base.cancelAt, g.base.cancel = 1, cancel
			}
			if outcome == "journal failure" {
				j.failKind = "run_checkpoint"
			}
			r := fixture(g.base, &executor{}, j)
			r.Generator = g
			_, err := r.Run(ctx, "request", approve, emit)
			if (err == nil) != (outcome == "complete") {
				t.Fatalf("outcome=%s err=%v", outcome, err)
			}
			if g.active || g.created != 1 || g.released != 1 {
				t.Fatal("request resources were released early, retained or released twice")
			}
			if outcome == "complete" {
				if _, err := r.Run(ctx, "second request", approve, emit); err != nil || g.created != 2 || g.released != 2 {
					t.Fatalf("next request reused the previous scope: %v", err)
				}
			}
		})
	}
}
