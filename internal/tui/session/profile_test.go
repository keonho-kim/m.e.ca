// Purpose: Verify model settings expose the applied profile and distinguish unidentified connections.
// Pattern: Terminal presentation tests.
// Usage: go test ./internal/tui/session -run TestModelProfileDisplay.
// Related: internal/tui/session/render_menu.go, internal/provider/connection/models.go.
package session

import (
	"strings"
	"testing"
)

func TestModelProfileDisplay(t *testing.T) {
	a := fixture(t)
	a.selection.Model = "team-coder"
	a.selection.Profile = "ornith-1.5-9b"
	a.openModel()
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.Contains(got, "프로파일: ornith-1.5-9b") {
		t.Fatalf("profile missing: %q", got)
	}
	a.selection.Profile = "generic"
	a.render()
	if got := screenLine(a.screen, 0, 80); !strings.Contains(got, "미식별 / 일반") {
		t.Fatalf("generic profile missing: %q", got)
	}
}
