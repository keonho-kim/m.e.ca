// Purpose: Resume a saved cooperative conversation with the same JSONL/text output contracts.
// Pattern: Noninteractive application use case.
// Usage: cmd/meca selects this path only when the chosen conversation has team mode enabled.
// Related: internal/commands/teamrequest/request.go, internal/commands/agentrequest/approval.go, internal/commands/headless/output.go.
package headless

import (
	"context"
	"io"

	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/commands/teamrequest"
)

func RunTeam(ctx context.Context, service *teamrequest.Service, options agentrequest.Options, writer io.Writer, jsonMode bool) error {
	view := newOutput(writer, jsonMode, options.Vault.Redact)
	if err := view.begin(options.Session.ID, options.Session.ProjectRoot); err != nil {
		return err
	}
	result, err := service.Run(ctx, options, agentrequest.RejectApproval, view.emit)
	return view.finish(result.Status, result.Text, err)
}
