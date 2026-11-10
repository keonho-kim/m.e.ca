// Purpose: Wire the upgrade command to authenticated release retrieval and report its observed result.
// Pattern: Composition Root.
// Usage: maintenance dispatches upgrade before session bootstrap or terminal setup.
// Related: cmd/meca/install.go, internal/commands/upgrade/upgrade.go, internal/commands/upgrade/credential.go.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/keonho-kim/meca/internal/commands/upgrade"
	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/distribution"
)

func upgradeCurrent(ctx context.Context, home string, kind distribution.Kind) error {
	fmt.Fprintln(os.Stdout, catalog.Text(catalog.UpgradeCheckingDistribution, kind))
	client, err := upgrade.NewClient(upgrade.Credential(ctx))
	if err != nil {
		return err
	}
	defer client.Close()
	result, err := upgrade.Run(ctx, home, client, kind)
	if err != nil {
		return err
	}
	if result.AlreadyCurrent {
		fmt.Fprintln(os.Stdout, catalog.Text(catalog.UpgradeDistributionCurrent, kind, result.Version))
	} else {
		fmt.Fprintln(os.Stdout, catalog.Text(catalog.UpgradeDistributionComplete, kind, result.Version))
	}
	fmt.Fprintln(os.Stdout, catalog.Text(catalog.UpgradePromptsRefreshed))
	return nil
}
