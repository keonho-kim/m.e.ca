// Purpose: Extract explicit skill names from current user text outside code, quotes and URLs.
// Pattern: Pure bounded input parser.
// Usage: Request and steering admission call Mentions once for each new user message.
// Related: internal/skills/instruction/mentions_test.go, internal/agent/run/skills.go.
package instruction

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func Mentions(text string) []string {
	if !strings.ContainsRune(text, '#') {
		return nil
	}
	var found []string
	seen := make(map[string]bool)
	fence := ""
	for line := range strings.SplitSeq(text, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			marker := trim[:3]
			if fence == "" {
				fence = marker
			} else if fence == marker {
				fence = ""
			}
			continue
		}
		if fence != "" || strings.HasPrefix(trim, ">") || strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			continue
		}
		var quote byte
		ticks := 0
		for i := 0; i < len(line); i++ {
			c := line[i]
			if c == '\\' {
				i++
				continue
			}
			if c == '`' && quote == 0 {
				n := 1
				for i+n < len(line) && line[i+n] == '`' {
					n++
				}
				if ticks == 0 {
					ticks = n
				} else if ticks == n {
					ticks = 0
				}
				i += n - 1
				continue
			}
			if ticks != 0 {
				continue
			}
			if quote != 0 {
				if c == quote {
					quote = 0
				}
				continue
			}
			previous, _ := utf8.DecodeLastRuneInString(line[:i])
			boundary := i == 0 || unicode.IsSpace(previous) || strings.ContainsRune("([{:;,", previous)
			if (c == '"' || c == '\'') && boundary {
				quote = c
				continue
			}
			if c != '#' || !boundary {
				continue
			}
			end := i + 1
			for end < len(line) && (line[end] >= 'a' && line[end] <= 'z' || line[end] >= '0' && line[end] <= '9' || line[end] == '-') {
				end++
			}
			name := line[i+1 : end]
			qualified := false
			if end < len(line) && line[end] == ':' && (name == "project" || name == "meca" || name == "shared") {
				start := end + 1
				end = start
				for end < len(line) && (line[end] >= 'a' && line[end] <= 'z' || line[end] >= '0' && line[end] <= '9' || line[end] == '-') {
					end++
				}
				if ValidName(line[start:end]) {
					name, qualified = name+":"+line[start:end], true
				} else {
					continue
				}
			}
			if end < len(line) {
				next, _ := utf8.DecodeRuneInString(line[end:])
				if unicode.IsLetter(next) || unicode.IsDigit(next) || strings.ContainsRune("_:/\\", next) {
					i = end
					continue
				}
			}
			if (qualified || ValidName(name) && strings.Trim(name, "0123456789") != "") && !seen[name] {
				seen[name] = true
				found = append(found, name)
			}
			i = max(i, end-1)
		}
	}
	return found
}
