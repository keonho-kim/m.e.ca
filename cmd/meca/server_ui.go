//go:build !tui

// Purpose: Announce the embedded UI and open one local interactive browser.
// Pattern: Process presentation adapter.
// Usage: The server ready callback invokes announceUI after the listener is bound.
// Related: cmd/meca/server.go, internal/diagnostics/catalog/server_presentation.go.
package main

import (
	"context"
	"fmt"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"golang.org/x/term"
	"net"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func announceUI(ctx context.Context, address string) {
	fmt.Fprintln(os.Stdout, diag.Text(diag.UIReady, address+"/ui/"))
	u, err := url.Parse(address)
	if err != nil || !net.ParseIP(u.Hostname()).IsLoopback() || !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := exec.CommandContext(ctx, name, address+"/ui/").Run(); err != nil {
		diagnostics.Log(ctx, diag.UIOperationFailed, "component", "browser", "error_code", diag.OperationFailed)
	}
}
