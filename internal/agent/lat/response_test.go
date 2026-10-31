// Purpose: Verify advisory findings, retry replacement and read-only review failure semantics.
// Pattern: Scripted model boundary tests.
// Usage: go test ./internal/agent/lat.
// Related: internal/agent/lat/response.go.
package lat

import (
	"context"
	"errors"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestReviewResponse(t *testing.T) {
	for _, test := range []struct {
		name         string
		events       []modelapi.Event
		clean, fails bool
		want         string
	}{
		{"clean", []modelapi.Event{{Delta: "NO_FINDINGS", Done: true}}, true, false, "NO_FINDINGS"},
		{"finding", []modelapi.Event{{Delta: "- api.py: missing route; add the requested endpoint.", Done: true}}, false, false, "missing route"},
		{"retry", []modelapi.Event{{Delta: "incorrect partial finding"}, {Retry: &modelapi.Retry{}}, {Delta: "NO_FINDINGS", Done: true}}, true, false, "NO_FINDINGS"},
		{"unfinished", []modelapi.Event{{Delta: "NO_FINDINGS"}}, false, true, ""},
		{"empty", []modelapi.Event{{Thinking: "reasoning alone", Done: true}}, false, true, ""},
		{"tool", []modelapi.Event{{Calls: []toolapi.Call{{Name: "write_file"}}, Done: true}}, false, true, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := Review(context.Background(), func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
				for _, event := range test.events {
					if err := emit(event); err != nil {
						return err
					}
				}
				return nil
			}, modelapi.Request{})
			if (err != nil) != test.fails || result.Clean != test.clean || !strings.Contains(result.Feedback, test.want) {
				t.Fatalf("review=%+v error=%v", result, err)
			}
		})
	}
}

func TestCancelledReviewDoesNotReturnPartialFinding(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result, err := Review(ctx, func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		if err := emit(modelapi.Event{Delta: "unfinished"}); err != nil {
			return err
		}
		cancel()
		return ctx.Err()
	}, modelapi.Request{})
	if !errors.Is(err, context.Canceled) || result.Feedback != "" || result.Clean {
		t.Fatalf("partial review escaped cancellation: %+v %v", result, err)
	}
}
