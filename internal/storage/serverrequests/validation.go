// Purpose: Parse durable compatible identities, terminal outcomes and immutable request metadata.
// Pattern: Persistence-boundary validation.
// Usage: Response reservation, update and read validate the same domain values.
// Related: internal/storage/serverrequests/responses.go, internal/commands/server/response.go.
package serverrequests

import (
	"encoding/hex"
	"fmt"
	"github.com/keonho-kim/meca/internal/commands/server"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"strings"
)

func validResponseID(id string) bool {
	if len(id) != 69 || !strings.HasPrefix(id, "resp_") {
		return false
	}
	_, err := hex.DecodeString(id[5:])
	return err == nil
}

func checkResponse(value server.Response) error {
	if !validResponseID(value.ID) || len(value.Scope) != 64 || len(value.Digest) != 64 || value.Created < 1 || value.Begin < 0 || value.Through < 0 || value.Session != "" && !sessionstate.ValidID(value.Session) {
		return fmt.Errorf("invalid response metadata; contents preserved")
	}
	if _, err := hex.DecodeString(value.Scope + value.Digest); err != nil {
		return fmt.Errorf("invalid response digest")
	}
	if value.Kind != "chat" && value.Kind != "response" {
		return fmt.Errorf("invalid response kind")
	}
	switch value.Status {
	case "queued", "in_progress", "completed", "failed", "cancelled", "incomplete":
	default:
		return fmt.Errorf("invalid response status")
	}
	if value.Previous != "" && !validResponseID(value.Previous) || value.Next != "" && !validResponseID(value.Next) {
		return fmt.Errorf("invalid response continuation")
	}
	if value.Task != "" {
		if value.Task != "task_"+value.ID[5:] || value.Session == "" {
			return fmt.Errorf("invalid response task")
		}
		if _, err := hex.DecodeString(value.Task[5:]); err != nil {
			return fmt.Errorf("invalid response task digest")
		}
	}
	if value.TaskStatus != "" && value.TaskStatus != "answered" && value.TaskStatus != "stopped" && value.TaskStatus != "interrupted" {
		return fmt.Errorf("invalid response task status")
	}
	if value.Status == "in_progress" || value.Status == "completed" || value.Status == "cancelled" || value.Status == "incomplete" {
		if value.Task == "" || value.Session == "" {
			return fmt.Errorf("response execution identity is missing")
		}
	}
	if value.Status != "queued" && value.Status != "in_progress" {
		if value.Task != "" && value.TaskStatus == "" {
			return fmt.Errorf("terminal response task status is missing")
		}
		if value.Completed < value.Created || value.Through < value.Begin {
			return fmt.Errorf("invalid terminal response boundaries")
		}
		if err := value.End.Check(); err != nil {
			return err
		}
	}
	if value.Next != "" && (value.Kind != "response" || !value.Stored || value.Status != "completed") {
		return fmt.Errorf("invalid continuation head")
	}
	if value.Kind == "chat" && (value.Background || value.Previous != "") {
		return fmt.Errorf("invalid chat response settings")
	}
	if value.Usage != nil {
		return value.Usage.CheckTotals()
	}
	return nil
}
