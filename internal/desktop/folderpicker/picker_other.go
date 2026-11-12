//go:build !darwin && !linux

// Purpose: Explicitly fall back to the server folder browser on unsupported desktops.
// Pattern: Platform adapter.
// Usage: Pick selects this implementation outside supported distribution targets.
// Related: internal/desktop/folderpicker/picker.go.
package folderpicker

import (
	"context"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func pick(context.Context) (string, error) {
	return "", fault.New(diag.Unavailable, diag.FolderPickerUnavailable)
}
