// Purpose: Distinguish image acceptance, output exhaustion, streamed rejection and cancellation.
// Pattern: Scripted generation contract tests.
// Usage: go test ./internal/provider/diagnostic -run TestImage.
// Related: internal/provider/diagnostic/image.go, internal/provider/transport/image.go.
package diagnostic

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/transport"
	"testing"
)

func TestImageProbeClassifications(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		generate   ImageGenerator
	}{
		{"arbitrary-answer", "supported", func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
			if len(r.Tools) != 0 || len(r.Messages) != 1 || len(r.Messages[0].Images) != 1 || r.Thinking != nil || r.MaxOutput != 32 {
				t.Fatal("probe contains unrelated work or controls")
			}
			return emit(modelapi.Event{Delta: "unrelated content", Done: true})
		}},
		{"length-after-thinking", "supported", func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
			if err := emit(modelapi.Event{Thinking: "draft"}); err != nil {
				return err
			}
			return &modelapi.OutputLimitError{Reason: "token budget"}
		}},
		{"stream-unsupported", "unsupported", func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
			return transport.ParseError(0, []byte(`{"error":{"code":"image_input_not_supported","message":"text only"}}`), "")
		}},
		{"invalid-image-is-not-unsupported", "unknown", func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
			return transport.ParseError(400, []byte(`{"error":{"message":"invalid image encoding"}}`), "")
		}},
		{"authentication", "unknown", func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
			return transport.ParseError(401, []byte(`{"error":{"message":"not authorized"}}`), "")
		}},
		{"empty-stream", "unknown", func(context.Context, modelapi.Request, func(modelapi.Event) error) error { return nil }},
		{"retry-clears-acceptance", "unknown", func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
			if err := emit(modelapi.Event{Done: true}); err != nil {
				return err
			}
			return emit(modelapi.Event{Retry: &modelapi.Retry{Attempt: 2}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := CheckImage(context.Background(), tc.generate, "m")
			if err != nil || result.State != tc.want {
				t.Fatalf("state=%q err=%v", result.State, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := CheckImage(ctx, func(ctx context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error { return ctx.Err() }, "m")
	if err != context.Canceled {
		t.Fatal("caller cancellation was swallowed", err)
	}
}
