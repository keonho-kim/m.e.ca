// Purpose: Present only commands available in the compiled distribution.
// Pattern: Command presentation adapter.
// Usage: Startup's help action calls printHelp before opening application state.
// Related: cmd/meca/main.go, internal/distribution/variants.go, internal/diagnostics/catalog/distribution.go.
package main

import (
	"fmt"
	"os"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/distribution"
)

func printHelp() {
	fmt.Fprintln(os.Stdout, catalog.Text(catalog.MecaHelpCommon, distribution.Name))
	if distribution.Name == distribution.Full {
		fmt.Fprintln(os.Stdout, catalog.Text(catalog.MecaHelpServer))
	}
	fmt.Fprintln(os.Stdout, catalog.Text(catalog.MecaHelpMaintenance))
}
