// Purpose: Fail approval requests explicitly when a caller has no interactive approval channel.
// Pattern: Noninteractive policy boundary.
// Usage: Headless commands and compatible HTTP submissions share RejectApproval.
// Related: internal/commands/headless/run.go, internal/commands/server/execution.go.
package agentrequest

import (
	"context"
	"errors"

	"github.com/keonho-kim/meca/internal/agent/tool"
)

var ErrApprovalRequired = errors.New("approval required in noninteractive mode; no operation was approved")

func RejectApproval(context.Context, tool.Approval) (bool, error) {
	return false, ErrApprovalRequired
}
