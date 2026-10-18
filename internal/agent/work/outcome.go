// Purpose: Define terminal work outcomes and validate their persisted identities.
// Pattern: Typed result contract.
// Usage: Run checkpoints and recovery preserve the terminal cause.
// Related: internal/agent/run/checkpoint.go, internal/agent/tool/reason.go.
package work

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/v-code/internal/agent/tool"
)

type Status string

const (
	Completed   Status = "completed"
	Failed      Status = "failed"
	Interrupted Status = "interrupted"
	Denied      Status = "denied"
	Exhausted   Status = "exhausted"
)

type End struct {
	Status Status `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (e End) Check() error {
	switch e.Status {
	case Completed, Failed, Interrupted, Denied, Exhausted:
	default:
		return fmt.Errorf("invalid work terminal status")
	}
	if e.Reason != "" && tool.CheckReason(e.Reason) != nil {
		return fmt.Errorf("invalid terminal reason")
	}
	return nil
}

func ValidID(id string) bool {
	return id != "" && len(id) <= 128 && utf8.ValidString(id) && !strings.ContainsAny(id, "\x00\r\n")
}
