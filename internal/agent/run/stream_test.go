// Purpose: Verify response bodies reach the UI before generation finishes without leaking native tool arguments.
// Pattern: Task streaming contract tests.
// Usage: go test ./internal/agent/run -run TestStreaming.
// Related: internal/agent/run/generate.go, internal/agent/tool/arguments.go.
package run

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"strings"
	"testing"
)

type chunkGenerator struct {
	chunks []string
	after  func(int)
	cancel context.CancelFunc
}

func (g *chunkGenerator) Generate(ctx context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
	for i, chunk := range g.chunks {
		if err := emit(modelapi.Event{Delta: chunk, Streaming: true}); err != nil {
			return err
		}
		if g.after != nil {
			g.after(i)
		}
	}
	if g.cancel != nil {
		g.cancel()
		return ctx.Err()
	}
	return emit(modelapi.Event{Done: true, Streaming: true})
}

func TestStreamingAnswerArrivesBeforeDone(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	var partial string
	g := &chunkGenerator{chunks: []string{"H", "el", "lo", " world"}}
	g.after = func(i int) {
		if i == 0 && partial != "H" {
			t.Fatalf("protocol head exposed: %q", partial)
		}
		if i == 1 && partial != "Hel" {
			t.Fatalf("body not delivered during generation: %q", partial)
		}
		if i == 2 && partial != "Hello" {
			t.Fatalf("body not progressive: %q", partial)
		}
	}
	r.Generator = g
	result, err := r.Run(context.Background(), "hello", false, approve, func(event Event) error {
		if event.Kind == "stream" {
			partial = event.Text
		}
		return nil
	})
	if err != nil || result.Text != "Hello world" || partial != "Hello world" {
		t.Fatalf("answer: %+v %q %v", result, partial, err)
	}
}

func TestStreamingCancellationPreservesOnlyMaskedBody(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	j := &journal{}
	r := fixture(&generator{}, &executor{}, j)
	vault := &credentials.Vault{}
	vault.Remember("provider-secret")
	r.Redact, r.MaskStream = vault.Redact, vault.Stream
	r.Generator = &chunkGenerator{chunks: []string{"hello provider-", "secret"}, cancel: cancel}
	var partial string
	_, err := r.Run(ctx, "hello", false, approve, func(event Event) error {
		if event.Kind == "stream" {
			partial = event.Text
			if strings.Contains(partial, "provider-") || strings.Contains(partial, "DONE") {
				t.Fatalf("unsafe partial: %q", partial)
			}
		}
		return nil
	})
	if err == nil || partial != "hello [redacted]" {
		t.Fatalf("partial=%q err=%v", partial, err)
	}
	last := j.messages[len(j.messages)-1]
	if last.Role != "assistant" || last.Content != partial {
		t.Fatalf("interrupted body not preserved: %+v", last)
	}
}

func TestStreamingPreservesReportAfterChanges(t *testing.T) {
	s := &execution{changed: true}
	if got := s.responseBody("Changed file; tests not run"); got != "Changed file; tests not run" {
		t.Fatalf("report withheld: %q", got)
	}
}

func TestStreamingPlanConversationArrivesBeforeDone(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	partial := ""
	g := &chunkGenerator{chunks: []string{"Hel", "lo!"}}
	g.after = func(i int) {
		if i == 0 && partial != "Hel" {
			t.Fatalf("Plan reply did not stream: %q", partial)
		}
	}
	r.Generator = g
	result, err := r.Run(context.Background(), "hello", true, approve, func(event Event) error {
		if event.Kind == "stream" {
			partial = event.Text
		}
		return nil
	})
	if err != nil || result.Status != Answered || partial != "Hello!" {
		t.Fatalf("%+v partial=%q %v", result, partial, err)
	}
}
