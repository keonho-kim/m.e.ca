//go:build !darwin && !linux

// Purpose: Reject isolated Chromium rendering on unsupported host platforms.
// Pattern: Build-selected platform boundary.
// Usage: The Chromium renderer calls start only through the same contract.
// Related: internal/tools/browser/chromium/process_unix.go.
package chromium

import (
	"context"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func start(context.Context, string, string) (*pipe, func() error, error) {
	return nil, nil, fault.Errorf(diag.WebFetchBrowser)
}
