// Purpose: Read the Windows host clipboard from a local WSL session without Linux clipboard tools.
// Pattern: Windows PowerShell binary-pipe adapter.
// Usage: WSL detection precedes Wayland/X11 selection; no resident bridge is started.
// Related: internal/tui/clipboard/wsl_path.go, internal/tui/clipboard/process.go.
package clipboard

import (
	"context"
	"errors"
	"os/exec"

	"github.com/keonho-kim/meca/internal/agent/media"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
	native "golang.design/x/clipboard"
)

// The host may have to convert a bitmap to PNG. Reserve that transient memory
// before launching it, and release before the common codec reserves its budget.
func readWSL(ctx context.Context) (native.Image, error) {
	release, err := processmemory.AcquireDecode(ctx, int64(media.MaxPixels)*8+int64(media.MaxBytes)*2)
	if err != nil {
		return native.Image{}, err
	}
	defer release()
	path, err := findPowerShell(exec.LookPath, readWSLMounts)
	if err != nil {
		return native.Image{}, faults.New(diag.Unavailable, diag.ClipboardImageWSLUnavailable)
	}
	cmd := exec.CommandContext(ctx, path, "-NoProfile", "-NonInteractive", "-STA", "-Command", windowsImageScript())
	value, err := readProcess(ctx, cmd)
	if err != nil && !errors.Is(err, native.ErrNoData) && !errors.Is(err, native.ErrImageLimit) && ctx.Err() == nil {
		return value, faults.New(diag.Unavailable, diag.ClipboardImageWSLUnavailable)
	}
	return value, err
}
