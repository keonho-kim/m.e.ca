// Purpose: Parse one tool decision or one literal typed argument without interpreting prose as code.
// Pattern: Pure Function (model trust boundary).
// Usage: Used by the task loop, provider diagnostics and the local tool argument boundary.
// Related: internal/agent/tools.go, internal/agent/dialogue/dialogue_test.go.
package dialogue

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Kind string

const (
	Tool     Kind = "TOOL"
	Check    Kind = "CHECK"
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
	head, body, _ := strings.Cut(strings.TrimSpace(text), "\n")
	fields := strings.Fields(head)
	if len(fields) == 0 {
		return Decision{}, fmt.Errorf("empty decision")
	}
	kind := Kind(fields[0])
	switch kind {
	case Tool, Check:
		if len(fields) != 2 || strings.TrimSpace(body) != "" {
			return Decision{}, fmt.Errorf("reply with exactly %s <tool_name>; arguments are requested separately", kind)
		}
		for _, r := range fields[1] {
			if !(r >= 'a' && r <= 'z' || r == '_') {
				return Decision{}, fmt.Errorf("invalid tool name")
			}
		}
		return Decision{Kind: kind, Name: fields[1]}, nil
	case Done, Wait, Continue:
		if len(fields) != 1 || strings.TrimSpace(body) == "" {
			return Decision{}, fmt.Errorf("%s requires a summary on the next line", kind)
		}
		return Decision{Kind: kind, Text: strings.TrimSpace(body)}, nil
	default:
		return Decision{}, fmt.Errorf("reply with TOOL, CHECK, DONE, WAIT or CONTINUE")
	}
}
func ParseArgument(p agent.ToolParameter, text string) (json.RawMessage, error) {
	if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return nil, fmt.Errorf("%s has invalid encoding", p.Name)
	}
	if p.Kind == agent.Integer {
		text = strings.TrimSpace(text)
		if text == "" && !p.Required {
			text = "0"
		}
		value, err := strconv.Atoi(text)
		if err != nil || value < p.Min || value > p.Max {
			return nil, fmt.Errorf("%s requires one integer from %d to %d", p.Name, p.Min, p.Max)
		}
		return json.Marshal(value)
	}
	if p.Kind == agent.LiteralPath {
		// A text response may end its single path line with LF or CRLF.
		text = strings.TrimRight(text, "\r\n")
		trimmed := strings.TrimSpace(text)
		if strings.ContainsAny(text, "\r\n") {
			return nil, fmt.Errorf("%s requires one plain path line", p.Name)
		}
		if json.Valid([]byte(trimmed)) && (strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "\"")) {
			return nil, fmt.Errorf("%s requires one literal path, not a JSON wrapper", p.Name)
		}
	}
	if p.Kind != agent.Text && p.Kind != agent.LiteralPath {
		return nil, fmt.Errorf("unsupported parameter kind")
	}
	if len(text) > p.MaxBytes || !p.AllowEmpty && strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("%s must be valid literal text within %d bytes", p.Name, p.MaxBytes)
	}
	return json.Marshal(text)
}
func ValidateArgument(p agent.ToolParameter, value json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return fmt.Errorf("%s cannot be null", p.Name)
	}
	var text string
	if p.Kind == agent.Integer {
		var number int
		if err := json.Unmarshal(value, &number); err != nil {
			return err
		}
		text = strconv.Itoa(number)
	} else {
		if err := json.Unmarshal(value, &text); err != nil {
			return err
		}
	}
	_, err := ParseArgument(p, text)
	return err
}
