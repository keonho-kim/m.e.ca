// Purpose: Archive sanitized command captures and return small, re-readable output excerpts.
// Pattern: Session-owned output projection.
// Usage: Command execution stores large captures until Executor.Close releases them.
// Related: internal/tools/local/command.go, internal/tools/local/executor.go.
package local

import (
	"os"
	"unicode/utf8"
)

const outputExcerptBytes = 32 * 1024

func (e *Executor) commandOutput(id, text, stream string) (string, string, error) {
	if len(text) <= outputExcerptBytes {
		return text, "", nil
	}
	path, err := e.commandPath(id, "."+stream+".log")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		return "", "", err
	}
	return commandExcerpt(text), path, nil
}

func commandExcerpt(text string) string {
	if len(text) <= outputExcerptBytes {
		return text
	}
	head, tail := outputExcerptBytes/2, len(text)-outputExcerptBytes/2
	for head > 0 && !utf8.RuneStart(text[head]) {
		head--
	}
	for tail < len(text) && !utf8.RuneStart(text[tail]) {
		tail++
	}
	return text[:head] + "\n[output excerpt; inspect the captured log path for omitted text]\n" + text[tail:]
}
