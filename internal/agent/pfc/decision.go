// Purpose: Parse a JSON PFC entry or an explicitly framed user-facing decision.
// Pattern: Pure Function at the model trust boundary.
// Usage: The agent loop and connection diagnostic call ParseDecision on a complete reply.
// Related: internal/agent/pfc/decision_test.go, internal/agent/run/run.go, internal/provider/diagnostic/dialogue.go.
package pfc

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type Kind string

const (
	Tool     Kind = "TOOL"
	Done     Kind = "DONE"
	Wait     Kind = "WAIT"
	Continue Kind = "CONTINUE"
)

type Decision struct {
	Kind       Kind
	Name, Text string
}

func ParseDecision(text string) (Decision, error) {
	if len(text) > 16384 || !utf8.ValidString(text) {
		return Decision{}, fmt.Errorf("invalid decision size or encoding")
	}
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "{") {
		return parseEntry(text)
	}
	head, body, _ := strings.Cut(text, "\n")
	fields := strings.Fields(head)
	if len(fields) == 0 {
		return Decision{}, fmt.Errorf("empty decision")
	}
	kind := Kind(fields[0])
	switch kind {
	case Done, Wait, Continue:
		if len(fields) != 1 || strings.TrimSpace(body) == "" {
			return Decision{}, fmt.Errorf("%s requires a summary on the next line", kind)
		}
		return Decision{Kind: kind, Text: strings.TrimSpace(body)}, nil
	default:
		return Decision{}, fmt.Errorf(`reply with {"tool":"tool_name"} for PFC, or DONE, WAIT or CONTINUE followed by a newline and text`)
	}
}

func parseEntry(text string) (Decision, error) {
	invalid := func() (Decision, error) {
		return Decision{}, fmt.Errorf(`PFC entry requires exactly one JSON object with only a nonempty "tool" string; no arguments, duplicate keys or surrounding text`)
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return invalid()
	}
	if key, err := decoder.Token(); err != nil || key != "tool" {
		return invalid()
	}
	var name string
	if err := decoder.Decode(&name); err != nil || name == "" {
		return invalid()
	}
	// Reading the closing delimiter now rejects both extra and duplicate fields.
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') {
		return invalid()
	}
	if _, err := decoder.Token(); err != io.EOF {
		return invalid()
	}
	for _, char := range name {
		if !(char >= 'a' && char <= 'z' || char == '_') {
			return Decision{}, fmt.Errorf("invalid tool name; copy a name from Available tools exactly")
		}
	}
	return Decision{Kind: Tool, Name: name}, nil
}
