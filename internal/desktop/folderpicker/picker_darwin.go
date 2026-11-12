// Purpose: Present the macOS folder chooser through the built-in AppleScript runtime.
// Pattern: Static-argument native desktop adapter.
// Usage: Pick calls this on macOS; cancellation closes its owned osascript process.
// Related: internal/desktop/folderpicker/picker.go.
package folderpicker

import (
	"context"
	"strings"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func pick(ctx context.Context) (string, error) {
	// Cancellation is a normal empty selection; paths never enter executable source.
	script := `try
return POSIX path of (choose folder with prompt "` + diag.Text(diag.FolderPickerTitle) + `")
on error number -128
return ""
end try`
	path, err := run(ctx, "/usr/bin/osascript", "-e", script)
	return strings.TrimSuffix(path, "\n"), err
}
