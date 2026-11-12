// Purpose: Keep large queued requests editable without duplicating them in transcript rows.
// Pattern: Reservation projection memory regression.
// Usage: go test ./internal/tui/composer/queue -run TestQueuedPreviewByteBudget.
// Related: internal/tui/composer/queue/projection.go, internal/tui/composer/queue/state.go.
package queue

import (
	"strings"
	"testing"

	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestQueuedPreviewByteBudgetPreservesSource(t *testing.T) {
	body := strings.Repeat("line\n", 10000)
	requests := []requestinput.Request{requestinput.Capture(body, nil)}
	var projection Projection
	projection.Prepare(requests, locale.English, 0, func(text string) string { return text })
	if len(projection.Entries[0].Block.Text) > 4096 || requests[0].Text != body {
		t.Fatal("queue display duplicates or changes the large editable source")
	}
}
