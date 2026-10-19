// Purpose: Deliver selected visible text to the terminal clipboard with a local macOS fallback.
// Pattern: Clipboard I/O adapter with owned cancellable worker.
// Usage: Selection copy calls copySelection; receive handles its asynchronous completion.
// Related: internal/tui/session/text_selection.go, internal/tui/session/events.go, internal/tui/session/app.go.
package session

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func (a *app) copySelection(text string) {
	a.screen.SetClipboard([]byte(text))
	a.stopNotice = a.language.Text("클립보드에 복사를 요청했습니다")
	// OSC 52 targets the user's terminal even over SSH. Do not modify a remote
	// Mac's desktop clipboard when the terminal is on another machine.
	if a.screen.HasClipboard() || runtime.GOOS != "darwin" || os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return
	}
	if a.clipboardBusy {
		return
	}
	a.clipboardBusy = true
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, "/usr/bin/pbcopy")
		command.Stdin = strings.NewReader(text)
		err := command.Run()
		a.post(a.ctx, notice{Clipboard: true, Err: err})
	}()
}
