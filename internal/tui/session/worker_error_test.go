// Purpose: Verify a completed worker error is rendered once, including after leaving a menu.
// Pattern: Terminal behavior test.
// Usage: go test ./internal/tui/session -run TestWorkerError.
// Related: internal/tui/session/worker_result.go, internal/tui/session/render_chat.go.
package session

import (
	"errors"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
)

func TestWorkerErrorDisplayedOnce(t *testing.T) {
	for _, page := range []string{"chat", "model"} {
		t.Run(page, func(t *testing.T) {
			a := screenFixture(t)
			a.page, a.worker = page, workerSetup
			a.receive(notice{Done: true, Err: errors.New("Connection diagnostic failed")})
			check := func() {
				t.Helper()
				a.render()
				var rows strings.Builder
				for row := 0; row < 24; row++ {
					rows.WriteString(screenLine(a.screen, row, 80))
				}
				if count := strings.Count(rows.String(), "Connection diagnostic failed"); count != 1 {
					t.Fatalf("error displayed %d times", count)
				}
			}
			check()
			if page != "chat" {
				a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
			}
			check()
		})
	}
}
