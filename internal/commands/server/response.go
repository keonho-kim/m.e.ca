// Purpose: Define durable compatible submission, response and original-output contracts.
// Pattern: Consumer-owned persistence and result contracts.
// Usage: Compatible submission and persistence exchange these values without transport DTOs.
// Related: internal/commands/server/submission.go, internal/storage/serverrequests/responses.go.
package server

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/work"
)

type InputMessage struct{ Role, Text string }
type Submission struct {
	Kind, Key, Previous string
	Messages            []InputMessage
	Stored, Background  bool
}

// Response stores only output boundaries. Text remains in the conversation journal.
type Response struct {
	ID, Scope, Digest, Kind       string
	Session, Task, Previous, Next string
	Status, Error, Code           string
	TaskStatus                    string
	Created, Completed            int64
	Begin, Through                int64
	Stored, Background, Team      bool
	End                           work.End
	Usage                         *model.TokenUsage
}

type ResponseStore interface {
	ReserveResponse(context.Context, Response) (Response, bool, error)
	ReadResponse(context.Context, string) (Response, error)
	SaveResponse(context.Context, Response) error
}

type OutputHistory interface {
	OutputPosition(context.Context, string) (int64, error)
	ResponseOutput(context.Context, string, int64, int64, int, func(int) error) (string, error)
}

type Submitted struct {
	Response  Response
	Cursor    string
	Duplicate bool
}
