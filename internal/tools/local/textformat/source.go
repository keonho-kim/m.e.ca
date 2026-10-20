// Purpose: Preserve source CRLF when redaction removes only carriage returns, never masked content.
// Pattern: Pure source transformation.
// Usage: Command output and mutation arguments call RedactSource before further use.
// Related: internal/tools/local/command.go, internal/tools/local/prepare.go, internal/tools/local/textformat/source_test.go.
package textformat

import "strings"

func RedactSource(source string, redact func(string) string) string {
	safe := redact(source)
	if safe == strings.ReplaceAll(source, "\r\n", "\n") {
		return source
	}
	return safe
}
