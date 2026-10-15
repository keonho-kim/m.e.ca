// Purpose: Verify terminal tool output shows line numbers while agent records retain full anchors.
// Pattern: Terminal projection contract test.
// Usage: go test ./internal/tui/session -run TestReadLineReferencesVisible.
// Related: internal/tui/transcript/tools.go, internal/tools/local/file/read.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestReadLineReferencesVisible(t *testing.T) {
	a := fixture(t)
	event := run.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "read_file", Arguments: []byte(`{"path":"a"}`)}, Output: []byte(`{"Content":"41#9a72d30f | first\n42#b6e108ca | second\n"}`)}
	a.receive(notice{Event: &event})
	a.render()
	var visible strings.Builder
	for y := 0; y < 24; y++ {
		visible.WriteString(screenLine(a.screen, y, 80))
	}
	for _, line := range []string{"41 | first", "42 | second"} {
		if !strings.Contains(visible.String(), line) {
			t.Fatalf("missing line reference: %s", visible.String())
		}
	}
	if strings.Contains(visible.String(), "9a72d30f") || strings.Contains(visible.String(), "b6e108ca") {
		t.Fatal("internal hash visible")
	}
	if !strings.Contains(string(event.Output), "41#9a72d30f") {
		t.Fatal("agent anchor changed")
	}

}
