// Purpose: Verify removed queue rows release both their raw source and rendered text.
// Pattern: Queue projection resource-lifetime regression test.
// Usage: go test ./internal/tui/composer/queue.
// Related: internal/tui/composer/queue/projection.go.
package queue

import (
	"github.com/keonho-kim/meca/internal/tui/composer"
	"testing"

	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestProjectionReleasesRemovedRows(t *testing.T) {
	var projection Projection
	redact := func(text string) string { return text }
	messages := []composer.Request{{Text: "first"}, {Text: "second"}, {Text: "third"}}
	projection.Prepare(messages, locale.English, 0, redact)
	retained := projection.Entries
	projection.Prepare(messages[:1], locale.English, 0, redact)
	for _, row := range retained[1:] {
		if row.source != "" || row.Block.Text != "" {
			t.Fatal("removed queue row retained source or display references")
		}
	}
	projection.Prepare(nil, locale.English, 0, redact)
	if projection.Entries != nil || retained[0].source != "" || retained[0].Block.Text != "" {
		t.Fatal("empty queue retained projection storage")
	}
}
