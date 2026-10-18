// Purpose: Preserve source CRLF when redaction removes only carriage returns, never masked content.
// Pattern: Pure source transformation.
// Usage: Shell observations, internal reads and mutation argument checks share RedactSource.
// Related: internal/tools/local/prepare.go, internal/tools/local/command.go, internal/tools/local/textedit/edit.go.
package textformat

import "strings"

func RedactSource(source string, redact func(string) string) string {
	safe := redact(source)
	if safe == strings.ReplaceAll(source, "\r\n", "\n") {
		return source
	}
	return safe
}
