// Purpose: Parse complete bounded tool-call identities and argument objects.
// Pattern: Tool trust boundary.
// Usage: The task runtime and tool adapter check calls before preparation.
// Related: internal/agent/tool/schema.go, internal/tools/local/catalog/arguments.go.

package tool

import (
	"encoding/json"
	"fmt"
	"unicode"
	"unicode/utf8"
)

const (
	MaxArgumentsBytes = 256 * 1024 * 1024
	MaxCalls          = 32
)

type Call struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (call Call) Check() error {
	if call.ID == "" || len(call.ID) > 128 || !utf8.ValidString(call.ID) {
		return fmt.Errorf("invalid Tool Call ID")
	}
	for _, char := range call.ID {
		if unicode.IsControl(char) {
			return fmt.Errorf("invalid Tool Call ID")
		}
	}
	if call.Name == "" || len(call.Name) > 128 {
		return fmt.Errorf("invalid tool name")
	}
	for _, char := range call.Name {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_') {
			return fmt.Errorf("invalid tool name")
		}
	}
	if len(call.Arguments) > MaxArgumentsBytes || !json.Valid(call.Arguments) {
		return fmt.Errorf("invalid or oversized Tool Call arguments")
	}
	if _, err := argumentObject(call.Arguments); err != nil {
		return err
	}
	return nil
}
