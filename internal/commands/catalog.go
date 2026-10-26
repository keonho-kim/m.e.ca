// Purpose: Own local command names and ranked search over names and descriptions.
// Pattern: Simple Module (fixed command catalog).
// Usage: The parser recognizes canonical names; the TUI queries Suggestions as the user types.
// Related: internal/commands/parse.go, internal/tui/session/hints.go, internal/commands/catalog_test.go.
package commands

import (
	"github.com/keonho-kim/meca/internal/searchmatch"
	"strings"
)

type Suggestion struct{ Name, Description string }

var catalog = []struct {
	name, description string
	kind              Kind
}{
	{"!model", "모델 선택 · 추론 설정", Model},
	{"!language", "언어 설정", Language},
	{"!permission", "승인 모드", Permission},
	{"!dir", "파일 탐색", Directory},
	{"!new", "새 세션", NewSession},
	{"!quit", "앱 종료", Exit},
}

func Suggestions(input string, describe func(string) string) []Suggestion {
	if !strings.HasPrefix(input, "!") {
		return nil
	}
	query := strings.ToLower(strings.TrimPrefix(input, "!"))
	var matches []Suggestion
	// Preserve short command prefixes as the first Enter choice, then include
	// substring and description matches in the fixed catalog order.
	for _, prefixPass := range []bool{true, false} {
		for _, command := range catalog {
			description := describe(command.description)
			name := strings.TrimPrefix(command.name, "!")
			prefix := strings.HasPrefix(name, query)
			if prefix != prefixPass {
				continue
			}
			if prefix || strings.Contains(name, query) || strings.Contains(strings.ToLower(description), query) {
				matches = append(matches, Suggestion{command.name, description})
			}
		}
	}
	if len(matches) == 0 && query != "" {
		for _, command := range catalog {
			if searchmatch.Subsequence(query, strings.TrimPrefix(command.name, "!")) {
				matches = append(matches, Suggestion{command.name, describe(command.description)})
			}
		}
	}
	return matches
}
