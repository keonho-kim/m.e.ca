// Purpose: Extract a safe missing-library name from native Linux loader diagnostics.
// Pattern: Narrow error translation without forwarding arbitrary browser stderr.
// Usage: Runtime verification and failed helper invocations report actionable system dependencies.
// Related: internal/tools/browser/runtime/managed.go, internal/tools/browser/capture.go.
package runtime

import (
	"regexp"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var loaderError = regexp.MustCompile(`error while loading shared libraries: ([A-Za-z0-9_+.\-]{1,128}\.so(?:\.[0-9]+)*): cannot open shared object file`)

func MissingLibrary(stderr []byte) error {
	match := loaderError.FindSubmatch(stderr)
	if match == nil {
		return nil
	}
	return fault.Errorf(diag.BrowserLibraryMissing, string(match[1]))
}
