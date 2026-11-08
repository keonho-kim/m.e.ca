// Purpose: Parse the shared portable skill-name constraints used by metadata and user references.
// Pattern: Pure identifier rule.
// Usage: Metadata decoding, selector resolution and mention parsing share ValidName.
// Related: internal/skills/format.go, internal/skills/reader.go, internal/skills/instruction/mentions.go.
package instruction

import "strings"

func ValidName(name string) bool {
	if len(name) == 0 || len(name) > 64 || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") || strings.Contains(name, "--") {
		return false
	}
	for _, c := range name {
		if c != '-' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}
