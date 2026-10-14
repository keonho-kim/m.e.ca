// Purpose: Retain bounded read snapshots and render short, snapshot-bound line anchors.
// Pattern: Owned snapshot store.
// Usage: File reads record complete visible lines; edits resolve anchors against this executor's reads.
// Related: internal/tools/local/anchors/edit.go, internal/tools/local/file/read.go, internal/tools/local/executor.go.
package anchors

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

const maxBytes = 4 * 1024 * 1024
const maxReads = 64

type snapshot struct {
	path, nonce   string
	first, weight int
	prefix        [32]byte
	lines         []string
}

// Store is owned by a local tool executor, not by the model or a global cache.
type Store struct {
	mu    sync.Mutex
	reads []snapshot
	bytes int
}

func split(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
func (s snapshot) anchor(index int) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d\x00%s", s.nonce, s.path, s.first+index, s.lines[index])))
	return fmt.Sprintf("%d#%x", s.first+index, digest[:4])
}
func (s *Store) Record(path string, first int, prefix [32]byte, text string) (string, error) {
	if path == "" || first < 1 || text == "" || !utf8.ValidString(text) {
		return "", fmt.Errorf("invalid read snapshot")
	}
	fingerprint := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%x\x00%s", path, first, prefix, text)))
	read := snapshot{path: path, nonce: fmt.Sprintf("%x", fingerprint), first: first, prefix: prefix, lines: split(text)}
	read.weight = len(path) + len(text) + len(read.lines)*16 + 256
	if read.weight > maxBytes || len(read.lines) > 32768 {
		return "", fmt.Errorf("read snapshot exceeds budget; request a smaller range")
	}
	var rendered strings.Builder
	for i, line := range read.lines {
		fmt.Fprintf(&rendered, "%s | %s", read.anchor(i), line)
		if !strings.HasSuffix(line, "\n") {
			rendered.WriteByte('\n')
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Identical observed evidence keeps its labels: repeated reads must not look
	// like progress merely because a new tool call was made.
	for i, old := range s.reads {
		if old.nonce == read.nonce && old.path == path && old.first == first && old.prefix == prefix && strings.Join(old.lines, "") == text {
			copy(s.reads[i:], s.reads[i+1:])
			s.reads[len(s.reads)-1] = old
			return rendered.String(), nil
		}
	}
	for len(s.reads) > 0 && (len(s.reads) >= maxReads || s.bytes+read.weight > maxBytes) {
		s.bytes -= s.reads[0].weight
		copy(s.reads, s.reads[1:])
		s.reads[len(s.reads)-1] = snapshot{}
		s.reads = s.reads[:len(s.reads)-1]
	}
	s.reads = append(s.reads, read)
	s.bytes += read.weight
	return rendered.String(), nil
}
func (s *Store) Clear() { s.mu.Lock(); defer s.mu.Unlock(); clear(s.reads); s.reads = nil; s.bytes = 0 }
