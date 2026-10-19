// Purpose: Archive sanitized command captures and return small, re-readable output excerpts.
// Pattern: Session-owned output projection.
// Usage: Command execution stores large captures until Executor.Close releases them.
// Related: internal/tools/local/command.go, internal/tools/local/executor.go.
package local

import (
	"errors"
	"os"
	"unicode/utf8"
)

const outputExcerptBytes = 32 * 1024

func (e *Executor) commandOutput(text, stream string) (string, string, error) {
	if len(text) <= outputExcerptBytes {
		return text, "", nil
	}
	f, err := os.CreateTemp(e.artifacts, stream+"-*.log")
	if err != nil {
		return "", "", err
	}
	_, writeErr := f.WriteString(text)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return "", "", err
	}
	head, tail := outputExcerptBytes/2, len(text)-outputExcerptBytes/2
	for head > 0 && !utf8.RuneStart(text[head]) {
		head--
	}
	for tail < len(text) && !utf8.RuneStart(text[tail]) {
		tail++
	}
	return text[:head] + "\n[output excerpt; inspect the captured log path for omitted text]\n" + text[tail:], f.Name(), nil
}
