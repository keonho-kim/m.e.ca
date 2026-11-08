// Purpose: Render safe terminal failures through the same registered diagnostic boundary.
// Pattern: Process exit presentation.
// Usage: main reports startup and cleanup failures before closing its stderr sink.
// Related: internal/diagnostics/fault/error.go, cmd/meca/main.go.
package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Exit(ctx context.Context, writer io.Writer, err error) int {
	if err == nil {
		return 0
	}
	Log(ctx, catalog.ProcessFailedEvent, "error_code", fault.Code(err))
	message := fault.PublicMessage(err, func(value string) string { return value })
	_, _ = fmt.Fprintln(writer, message)
	if errors.Is(err, context.Canceled) && fault.Code(err) == catalog.Canceled {
		return 130
	}
	return 1
}
