// Purpose: Own local command names and ranked search over names and descriptions.
// Pattern: Simple Module (fixed command catalog).
// Usage: The parser recognizes canonical names; the TUI queries Suggestions as the user types.
// Related: internal/commands/parse.go, internal/tui/session/hints.go, internal/commands/catalog_test.go.
package commands

import "strings"

type Suggestion struct{ Name, Description string }

var catalog = []struct {
	name, description string
	kind              Kind
}{
	{"!model", "모델 선택 · Thinking 설정", Model},
	{"!permission", "승인 정책", Permission},
	{"!dir", "파일 탐색", Directory},
	{"!new", "새 세션", NewSession},
	{"!quit", "앱 종료", Exit},
}

func Suggestions(input string) []Suggestion {
	if !strings.HasPrefix(input, "!") {
		return nil
	}
	query := strings.ToLower(strings.TrimPrefix(input, "!"))
	var matches []Suggestion
	// Preserve short command prefixes as the first Enter choice, then include
	// substring and description matches in the fixed catalog order.
	for _, prefixPass := range []bool{true, false} {
		for _, command := range catalog {
			name := strings.TrimPrefix(command.name, "!")
			prefix := strings.HasPrefix(name, query)
			if prefix != prefixPass {
				continue
			}
			if prefix || strings.Contains(name, query) || strings.Contains(strings.ToLower(command.description), query) {
				matches = append(matches, Suggestion{command.name, command.description})
			}
		}
	}
	return matches
}
