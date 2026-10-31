// Purpose: Collect one independent review while rejecting effects and incomplete replies.
// Pattern: Model response boundary.
// Usage: The run owner supplies generation with shared retry, cancellation and accounting.
// Related: internal/agent/lat/packet.go, internal/agent/run/lat.go.
package lat

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type Result struct {
	Feedback string
	Clean    bool
}

func Review(ctx context.Context, generate func(context.Context, modelapi.Request, func(modelapi.Event) error) error, request modelapi.Request) (Result, error) {
	var text strings.Builder
	done := false
	err := generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			text.Reset()
			done = false
		}
		if len(event.Calls) > 0 {
			return fmt.Errorf("LAT review attempted an unavailable tool")
		}
		if text.Len()+len(event.Delta) > 16*1024*1024 {
			return fmt.Errorf("LAT review response exceeds 16 MiB")
		}
		text.WriteString(event.Delta)
		done = done || event.Done
		return ctx.Err()
	})
	if err != nil {
		return Result{}, err
	}
	feedback := strings.TrimSpace(text.String())
	if !done || feedback == "" || !utf8.ValidString(feedback) {
		return Result{}, fmt.Errorf("LAT review did not produce a complete response")
	}
	return Result{Feedback: feedback, Clean: feedback == "NO_FINDINGS"}, nil
}
