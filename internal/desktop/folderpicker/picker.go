// Purpose: Own bounded native folder-picker subprocesses and normalize their selected path.
// Pattern: Desktop I/O adapter with caller-owned cancellation.
// Usage: The full binary injects Pick into the UI transport; TUI and the engine do not import it.
// Related: internal/desktop/folderpicker/picker_darwin.go, internal/desktop/folderpicker/picker_linux.go.
package folderpicker

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Pick(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "" {
		return "", fault.New(diag.Unavailable, diag.FolderPickerUnavailable)
	}
	path, err := pick(ctx)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		if fault.Code(err) == diag.Unavailable {
			return "", err
		}
		return "", fault.New(diag.Internal, diag.FolderPickerFailureCause, err)
	}
	return selectedFolder(path)
}

func selectedFolder(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if len(path) > 4096 || !utf8.ValidString(path) || strings.ContainsRune(path, 0) || !filepath.IsAbs(path) {
		return "", fault.New(diag.Internal, diag.FolderPickerFailed)
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fault.New(diag.Internal, diag.FolderPickerFailed)
	}
	return path, nil
}

type pathOutput struct{ body []byte }

func (out *pathOutput) Write(value []byte) (int, error) {
	if len(out.body)+len(value) > 16<<10 {
		return 0, io.ErrShortWrite
	}
	out.body = append(out.body, value...)
	return len(value), nil
}

func run(ctx context.Context, name string, arguments ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, arguments...)
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	// WSL's desktop process monitors this pipe, so it cannot outlive its proxy.
	input, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	defer input.Close()
	stop := context.AfterFunc(ctx, func() { _ = input.Close() })
	defer stop()
	var output pathOutput
	cmd.Stdout = &output
	err = cmd.Run()
	return string(output.body), err
}
