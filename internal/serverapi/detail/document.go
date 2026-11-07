// Purpose: Define an explicit public detail JSON schema that excludes private runtime and provider state.
// Pattern: Allowlisted wire document projection.
// Usage: The detail cache marshals these owned DTOs and masks JSON values before paging.
// Related: internal/serverapi/detail/event.go, docs/server.md.
package detail

import (
	"encoding/json"
	"fmt"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/redaction"
)

type document struct {
	Kind     string    `json:"kind"`
	Sequence int64     `json:"sequence,string,omitempty"`
	Message  *message  `json:"message,omitempty"`
	Event    *event    `json:"event,omitempty"`
	Approval *approval `json:"approval,omitempty"`
}

type message struct {
	ID         string `json:"id,omitempty"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	Incomplete bool   `json:"incomplete,omitempty"`
}

type approval struct {
	Preview string   `json:"preview"`
	Command *command `json:"command,omitempty"`
}

type command struct {
	Directory string    `json:"directory"`
	Findings  []finding `json:"findings"`
}

type finding struct {
	Command string   `json:"command"`
	Reasons []string `json:"reasons"`
	Targets []string `json:"targets"`
}

func marshalDocument(original server.DetailOriginal, redact func(string) string) ([]byte, error) {
	var value document
	switch {
	case original.Tool != nil:
		value.Kind, value.Event = "tool", publicEvent(*original.Tool)
	case original.History != nil:
		h := original.History
		value.Kind, value.Sequence = h.Kind, h.Sequence
		if h.Kind == "message" {
			m := h.Message
			value.Message = &message{m.ID, m.Role, m.Content, m.Incomplete}
		} else {
			value.Event = publicEvent(h.Event)
		}
	case original.Approval != nil:
		a := original.Approval
		value.Kind, value.Approval = "approval", &approval{Preview: a.Preview}
		if a.Command != nil {
			c := &command{Directory: a.Command.Directory}
			for _, f := range a.Command.Findings {
				c.Findings = append(c.Findings, finding{f.Command, f.Reasons, f.Targets})
			}
			value.Approval.Command = c
		}
	default:
		return nil, fmt.Errorf("detail has no visible original")
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return redaction.RedactJSON(body, redact)
}
