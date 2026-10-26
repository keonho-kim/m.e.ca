// Purpose: Verify terminal tool output shows line numbers while shell observations retain their original source.
// Pattern: Terminal projection contract test.
// Usage: go test ./internal/tui/session -run TestReadLineReferencesVisible.
// Related: internal/tui/transcript/tools.go, internal/tools/local/textedit/edit.go.
package session

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestReadLineReferencesVisible(t *testing.T) {
	a := fixture(t)
	event := runrecord.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"nl a"}`)}, Output: []byte(`{"Stdout":" 41 | first\n 42 | second\n","ExitCode":0,"Status":"exited"}`)}
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

}
