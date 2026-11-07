// Purpose: Identify immutable public conversation origins independently of tool JSON or display IDs.
// Pattern: Domain value with explicit relative-room references.
// Usage: Model messages, tool outcomes and journal events carry the same provenance.
// Related: internal/agent/team/channel/context.go, internal/storage/sqlite/team_memory.go.
package source

import (
	"fmt"
	"strings"
)

// Public resolves an empty Session against the executing peer's owned public room.
// Sequence zero denotes a local sanitized observation, without an original reference.
type Public struct {
	Session  string `json:"session,omitempty"`
	Sequence int64  `json:"sequence,omitempty"`
	Author   string `json:"author"`
	Kind     string `json:"kind"`
}

func (p Public) Check() error {
	if p.Sequence < 0 || p.Author == "" || len(p.Author) > 128 || len(p.Session) > 128 || strings.ContainsAny(p.Session+p.Author, "\r\n\x00") {
		return fmt.Errorf("invalid public provenance")
	}
	switch p.Kind {
	case "user_message", "team_message", "team_progress", "team_work", "team_discussion", "team_observation", "team_status":
	default:
		return fmt.Errorf("invalid public source kind")
	}
	if (p.Kind == "user_message") != (p.Author == "user") {
		return fmt.Errorf("public author and kind disagree")
	}
	return nil
}
