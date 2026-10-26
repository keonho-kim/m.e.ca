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
)

func upgradeCurrent(ctx context.Context, home string) error {
	fmt.Fprintln(os.Stdout, "M.E.CA 최신 릴리스를 확인합니다…")
	client, err := upgrade.NewClient(upgrade.Credential(ctx))
	if err != nil {
		return err
	}
	defer client.Close()
	result, err := upgrade.Run(ctx, home, client)
	if err != nil {
		return err
	}
	if result.AlreadyCurrent {
		fmt.Fprintln(os.Stdout, "이미 최신 릴리스입니다:", result.Version)
	} else {
		fmt.Fprintln(os.Stdout, "M.E.CA 업데이트 완료:", result.Version)
	}
	fmt.Fprintln(os.Stdout, "기본 프롬프트를 갱신했습니다.")
	return nil
}
