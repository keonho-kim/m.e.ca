// Purpose: Check serialized file-fact budgets without allocating a complete JSON snapshot.
// Pattern: Pure size calculation for a fixed domain value.
// Usage: Facts.Check validates the same 16 MiB contract before persistence.
// Related: internal/agent/tool/facts.go, internal/storage/sqlite/run_state.go.
package tool

import (
	"strconv"
	"unicode/utf8"
)

func factsBytes(f Facts) int {
	if len(f.Files) == 0 {
		return 2
	}
	n := len(`{"files":[]}`) + len(f.Files) - 1
	for _, file := range f.Files {
		n += len(`{"path":,"version":,"changed":,"stale":}`) + quotedBytes(file.Path) + quotedBytes(file.Version)
		n += boolBytes(file.Changed) + boolBytes(file.Stale)
		for _, field := range []struct{ name, value string }{{"observation", file.Observation}, {"source_id", file.SourceID}, {"observed_run", file.ObservedRun}} {
			if field.value != "" {
				n += len(field.name) + 4 + quotedBytes(field.value)
			}
		}
		if file.Revision != 0 {
			n += len(`,"revision":`) + len(strconv.Itoa(file.Revision))
		}
	}
	return n
}

func boolBytes(value bool) int {
	if value {
		return 4
	}
	return 5
}

func quotedBytes(text string) int {
	n := 2
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		switch {
		case r == '"' || r == '\\' || r == '\n' || r == '\r' || r == '\t' || r == '\b' || r == '\f':
			n += 2
		case r == utf8.RuneError && size == 1:
			n += 3
		case r < 32 || r == '<' || r == '>' || r == '&' || r == '\u2028' || r == '\u2029':
			n += 6
		default:
			n += size
		}
	}
	return n
}
