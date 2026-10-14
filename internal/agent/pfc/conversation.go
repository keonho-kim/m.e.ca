// Purpose: Distinguish an ordinary conversational answer from a complete JSON tool selection.
// Pattern: Pure parser at the conversation entry boundary.
// Usage: Initial conversation turns use ParseConversation; task turns retain ParseDecision.
// Related: internal/agent/pfc/decision.go, internal/agent/run/conversation.go.
package pfc

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func ParseConversation(text string) (Decision, error) {
	if len(text) > 16384 || !utf8.ValidString(text) {
		return Decision{}, fmt.Errorf("invalid conversation response size or encoding")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return Decision{}, fmt.Errorf("empty conversation response")
	}
	decision, err := ParseDecision(text)
	if err == nil && decision.Kind != Continue {
		return decision, nil
	}
	fields := strings.Fields(text)
	switch fields[0] {
	case "TOOL", "CHECK", "DONE", "WAIT", "CONTINUE", "GOAL:", "TASK":
		return Decision{}, fmt.Errorf("reply to the user directly or select one tool with a tool-only JSON object; do not output internal analysis")
	}
	if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		return Decision{}, fmt.Errorf("tool selection must be one tool-only JSON object; use DONE followed by a newline for a JSON answer")
	}
	return Decision{Kind: Done, Text: text}, nil
}
