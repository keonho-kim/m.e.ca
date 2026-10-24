// Purpose: Declare persistence capabilities used by the terminal session.
// Pattern: Consumer-owned repository contract.
// Usage: Options receives a repository wired by cmd/meca; the session does not own its database.
// Related: internal/storage/history/record.go, internal/agent/session/request.go.
package session

import (
	"context"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	tasksession "github.com/keonho-kim/meca/internal/agent/session"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/history"
)

type Store interface {
	tasksession.Store
	Selection(context.Context, string) (modelconn.Selection, error)
	SaveSelection(context.Context, string, modelconn.Selection) error
	FileSelections(context.Context, string) ([]string, error)
	SelectFile(context.Context, string, string, bool) error
	Timeline(context.Context, string, int64, int) ([]history.Record, int64, error)
	TokenUsage(context.Context, string) (modelapi.TokenUsage, error)
}
