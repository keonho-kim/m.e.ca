// Purpose: Define terminal work outcomes and validate their persisted identities.
// Pattern: Typed result contract.
// Usage: Run checkpoints and recovery preserve the terminal cause.
// Related: internal/agent/record/checkpoint.go, internal/agent/run/settle.go.
package work

import (
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Status string

const (
	Completed   Status = "completed"
	Failed      Status = "failed"
	Interrupted Status = diagcatalog.Interrupted
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
		return faults.Errorf(diagcatalog.WorkInvalidWorkTerminalStatus)
	}
	if len(e.Reason) > 32*1024*1024 || !utf8.ValidString(e.Reason) || strings.ContainsRune(e.Reason, 0) {
		return faults.Errorf(diagcatalog.WorkInvalidTerminalReason)
	}
	return nil
}

func ValidID(id string) bool {
	return id != "" && len(id) <= 128 && utf8.ValidString(id) && !strings.ContainsAny(id, "\x00\r\n")
}
