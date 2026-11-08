// Purpose: Parse durable compatible identities, terminal outcomes and immutable request metadata.
// Pattern: Persistence-boundary validation.
// Usage: Response reservation, update and read validate the same domain values.
// Related: internal/storage/serverrequests/responses.go, internal/commands/server/response.go.
package serverrequests

import (
	"encoding/hex"

	"strings"

	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func validResponseID(id string) bool {
	if len(id) != 69 || !strings.HasPrefix(id, "resp_") {
		return false
	}
	_, err := hex.DecodeString(id[5:])
	return err == nil
}

func checkResponse(value server.Response) error {
	if !validResponseID(value.ID) || len(value.Scope) != 64 || len(value.Digest) != 64 || value.Created < 1 || value.Begin < 0 || value.Through < 0 || !diagcatalog.ValidCode(value.Code) || value.Session != "" && !sessionstate.ValidID(value.Session) {
		return faults.Errorf(diagcatalog.ServerrequestsInvalidResponseMetadataContentsPreserved)
	}
	if _, err := hex.DecodeString(value.Scope + value.Digest); err != nil {
		return faults.Errorf(diagcatalog.ServerrequestsInvalidResponseDigest)
	}
	if value.Kind != "chat" && value.Kind != "response" {
		return faults.Errorf(diagcatalog.ServerrequestsInvalidResponseKind)
	}
	switch value.Status {
	case "queued", "in_progress", "completed", "failed", "cancelled", "incomplete":
	default:
		return faults.Errorf(diagcatalog.ServerrequestsInvalidResponseStatus)
	}
	if value.Previous != "" && !validResponseID(value.Previous) || value.Next != "" && !validResponseID(value.Next) {
		return faults.Errorf(diagcatalog.ServerrequestsInvalidResponseContinuation)
	}
	if value.Task != "" {
		if value.Task != "task_"+value.ID[5:] || value.Session == "" {
			return faults.Errorf(diagcatalog.ServerrequestsInvalidResponseTask)
		}
		if _, err := hex.DecodeString(value.Task[5:]); err != nil {
			return faults.Errorf(diagcatalog.ServerrequestsInvalidResponseTaskDigest)
		}
	}
	if value.TaskStatus != "" && value.TaskStatus != "answered" && value.TaskStatus != "stopped" && value.TaskStatus != diagcatalog.Interrupted {
		return faults.Errorf(diagcatalog.ServerrequestsInvalidResponseTaskStatus)
	}
	if value.Status == "in_progress" || value.Status == "completed" || value.Status == "cancelled" || value.Status == "incomplete" {
		if value.Task == "" || value.Session == "" {
			return faults.Errorf(diagcatalog.ServerrequestsResponseExecutionIdentityIsMissing)
		}
	}
	if value.Status != "queued" && value.Status != "in_progress" {
		if value.Task != "" && value.TaskStatus == "" {
			return faults.Errorf(diagcatalog.ServerrequestsTerminalResponseTaskStatusIsMissing)
		}
		if value.Completed < value.Created || value.Through < value.Begin {
			return faults.Errorf(diagcatalog.ServerrequestsInvalidTerminalResponseBoundaries)
		}
		if err := value.End.Check(); err != nil {
			return err
		}
	}
	if value.Next != "" && (value.Kind != "response" || !value.Stored || value.Status != "completed") {
		return faults.Errorf(diagcatalog.ServerrequestsInvalidContinuationHead)
	}
	if value.Kind == "chat" && (value.Background || value.Previous != "") {
		return faults.Errorf(diagcatalog.ServerrequestsInvalidChatResponseSettings)
	}
	if value.Usage != nil {
		return value.Usage.CheckTotals()
	}
	return nil
}
