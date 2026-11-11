// Purpose: Open the generated browser archive directly from immutable executable storage.
// Pattern: Standard ZIP filesystem adapter.
// Usage: New constructs the HTTP handler; source checkouts without a bundle reject --ui.
// Related: devtools/cmd/uiassets/bundle.go, internal/webui/assets.go.
package webui

import (
	"archive/zip"
	"io/fs"
	"strings"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var packedBundle string

func bundledFiles() (fs.FS, error) {
	if packedBundle == "" {
		return nil, fault.New(diag.FailedPrecondition, diag.UIBuildMissing)
	}
	return zip.NewReader(strings.NewReader(packedBundle), int64(len(packedBundle)))
}
