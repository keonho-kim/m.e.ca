// Purpose: Fail approval requests explicitly when a caller has no interactive approval channel.
// Pattern: Noninteractive policy boundary.
// Usage: Headless commands and compatible HTTP submissions share RejectApproval.
// Related: internal/commands/headless/run.go, internal/commands/server/execution.go.
package agentrequest

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrApprovalRequired = faults.Errorf(diagcatalog.AgentrequestApprovalRequiredInNoninteractiveModeNoOperation)

func RejectApproval(context.Context, tool.Approval) (bool, error) {
	return false, ErrApprovalRequired
}
