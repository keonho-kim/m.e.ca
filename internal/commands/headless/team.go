// Purpose: Resume a saved cooperative conversation with the same JSONL/text output contracts.
// Pattern: Noninteractive application use case.
// Usage: cmd/meca selects this path only when the chosen conversation has team mode enabled.
// Related: internal/commands/teamrequest/request.go, internal/commands/headless/output.go.
package headless

import (
	"context"
	"fmt"
	"io"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/commands/teamrequest"
)

func RunTeam(ctx context.Context, service *teamrequest.Service, options agentrequest.Options, writer io.Writer, jsonMode bool) error {
	view := newOutput(writer, jsonMode, options.Vault.Redact)
	if err := view.begin(options.Session.ID, options.Session.ProjectRoot); err != nil {
		return err
	}
	result, err := service.Run(ctx, options, noApproval, view.emit)
	return view.finish(result.Status, result.Text, err)
}
func noApproval(context.Context, toolapi.Approval) (bool, error) {
	return false, fmt.Errorf("approval required in noninteractive mode; no operation was approved")
}
