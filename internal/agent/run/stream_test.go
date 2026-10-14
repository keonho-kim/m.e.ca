// Purpose: Verify response bodies reach the UI before generation finishes without leaking tool pfc.
// Pattern: Task streaming contract tests.
// Usage: go test ./internal/agent/run -run TestStreaming.
// Related: internal/agent/run/generate.go, internal/agent/pfc/pfc.go.
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
	g := &chunkGenerator{chunks: []string{"DO", "NE\nHel", "lo", " world"}}
	g.after = func(i int) {
		if i == 0 && partial != "" {
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
	r.Generator = &chunkGenerator{chunks: []string{"DONE\nhello provider-", "secret"}, cancel: cancel}
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

func TestStreamingHidesProtocolAndUnapprovedCompletion(t *testing.T) {
	for _, test := range []struct {
		text, phase      string
		planning, worked bool
	}{
		{"TOOL read_file", "decision", false, false},
		{"CHECK execute_command", "decision", false, false},
		{"CONTINUE\nnext action", "decision", false, false},
		{"DONE\nfile content draft", "argument", false, false},
		{"DONE\nplan JSON", "decision", true, false},
		{"DONE\nclaim without verification", "decision", false, true},
		{"DONE extra\ninvalid head", "decision", false, false},
	} {
		s := task{checkpoint: Checkpoint{Phase: test.phase}, planning: test.planning, worked: test.worked}
		if got := s.responseBody(test.text); got != "" {
			t.Fatalf("protocol/unsafe completion exposed: %q", got)
		}
	}
}
