// Purpose: Select WSL or native clipboard input and map dependency errors to central diagnostics.
// Pattern: Concrete input adapter composition.
// Usage: ReadImage keeps clipboard access behind explicit TUI paste actions.
// Related: internal/tui/clipboard/worker.go, internal/tui/clipboard/wsl.go.
package clipboard

import (
	"context"
	"errors"
	"runtime"

	"github.com/keonho-kim/meca/internal/agent/media"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/host/wsl"
	native "golang.design/x/clipboard"
)

func readLocalImage(ctx context.Context) ([]byte, error) {
	var value native.Image
	var err error
	switch {
	case runtime.GOOS == "darwin":
		value, err = readNativeWorker(ctx)
	default:
		value, err = nativeSelection(ctx)
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, clipboardError(err)
	}
	if value.FileURL != "" {
		return readClipboardFile(ctx, value.FileURL)
	}
	return value.Data, nil
}

func nativeSelection(ctx context.Context) (native.Image, error) {
	if wsl.Available() {
		return readWSL(ctx)
	}
	return native.ReadImage(ctx, media.MaxBytes)
}

func clipboardError(err error) error {
	var failure *faults.Error
	if errors.As(err, &failure) {
		return err
	}
	if errors.Is(err, native.ErrNoData) {
		return faults.New(diag.InvalidArgument, diag.ClipboardImageUnavailable)
	}
	if errors.Is(err, native.ErrImageLimit) {
		return faults.New(diag.InvalidArgument, diag.ClipboardImageTooLarge)
	}
	if errors.Is(err, native.ErrUnavailable) || errors.Is(err, native.ErrUnsupported) {
		return faults.New(diag.Unavailable, diag.ClipboardImageNativeUnavailable)
	}
	return faults.New(diag.Unavailable, diag.ClipboardImageReadFailed)
}
