// Purpose: Own the six local command names used by parsing and prefix suggestions.
// Pattern: Simple Module (fixed command catalog).
// Usage: The parser recognizes names; the TUI queries Suggestions for the current prefix.
// Related: internal/commands/parse.go, internal/tui/session/hints.go, internal/commands/catalog_test.go.
package commands

import "strings"

type Suggestion struct{ Name, Description string }

var catalog = []struct {
	name, description string
	kind              Kind
}{
	{"!plan", "계획 모드", Plan},
	{"!model", "모델 선택 · Thinking 설정", Model},
	{"!permission", "승인 정책", Permission},
	{"!scope", "접근 범위", Scope},
	{"!dir", "파일 탐색", Directory},
	{"!quit", "앱 종료", Exit},
}

func Suggestions(prefix string) []Suggestion {
	var matches []Suggestion
	for _, command := range catalog {
		if strings.HasPrefix(command.name, prefix) {
			matches = append(matches, Suggestion{command.name, command.description})
		}
	}
	return matches
}
