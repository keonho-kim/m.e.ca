// Purpose: Parse complete bounded tool-call identities and argument objects.
// Pattern: Tool trust boundary.
// Usage: The task runtime and tool adapter check calls before preparation.
// Related: internal/agent/tool/schema.go, internal/tools/local/catalog/arguments.go.

package tool

import (
	"encoding/json"

	"unicode"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const (
	MaxArgumentsBytes = 256 * 1024 * 1024
	MaxCalls          = 32
)

type Call struct {
	// ExecutionID is assigned by the runtime, separately from the provider's ID.
	ExecutionID string          `json:"execution_id,omitempty"`
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Arguments   json.RawMessage `json:"arguments"`
}

// ExecutionKey uses the admitted execution identity. Candidates that never reach
// admission still have their provider identity for local validation feedback.
func (call Call) ExecutionKey() string {
	if call.ExecutionID != "" {
		return call.ExecutionID
	}
	return call.ID
}

// CheckIdentity validates call routing and byte bounds; schema parsing owns the argument object.
func (call Call) CheckIdentity() error {
	if call.ID == "" || len(call.ID) > 128 || !utf8.ValidString(call.ID) {
		return faults.Errorf(diagcatalog.ToolInvalidToolCallID)
	}
	for _, char := range call.ID {
		if unicode.IsControl(char) {
			return faults.Errorf(diagcatalog.ToolInvalidToolCallID)
		}
	}
	if call.Name == "" || len(call.Name) > 128 {
		return faults.Errorf(diagcatalog.ToolInvalidToolName)
	}
	for _, char := range call.Name {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_') {
			return faults.Errorf(diagcatalog.ToolInvalidToolName)
		}
	}
	if len(call.Arguments) > MaxArgumentsBytes {
		return faults.Errorf(diagcatalog.ToolInvalidOrOversizedToolCallArguments)
	}
	return nil
}

func (call Call) Check() error {
	if err := call.CheckIdentity(); err != nil {
		return err
	}
	if _, err := argumentObject(call.Arguments); err != nil {
		return err
	}
	return nil
}
