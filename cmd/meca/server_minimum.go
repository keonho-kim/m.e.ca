//go:build minimum

// Purpose: Reject server dispatch without importing server adapters into minimum binaries.
// Pattern: Build-selected executable boundary.
// Usage: The shared composition root retains this guarded dispatch contract.
// Related: cmd/meca/main.go, cmd/meca/server.go, internal/commands/startup/server.go.
package main

import (
	"context"

	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

func runServer(context.Context, startup.Mode, config.Config, string, string, *sqlite.Sessions) error {
	return fault.New(catalog.FailedPrecondition, catalog.DistributionFullRequired)
}
