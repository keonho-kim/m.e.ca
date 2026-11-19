// Purpose: Store unread UTF-8 observation bytes and reclaim each consumed block.
// Pattern: Forward-only artifact owned by one serialized tool session.
// Usage: Browser inspection and WebFetch keep their cursor identity and policy outside Body.
// Related: internal/tools/browser/paging.go, internal/tools/browser/webfetch/session.go.
package observation

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"unicode/utf8"
)

const BlockBytes = 128 << 10

type Body struct {
	path              string
	size, start, next int
	blocks            []int
}

func (b Body) Next() int     { return b.next }
func (b Body) Size() int     { return b.size }
func (b Body) Retained() int { return b.size - b.start }

// End returns a UTF-8 boundary at or before the requested byte limit.
func End(text string, limit int) int {
	end := min(limit, len(text))
	for end > 0 && end < len(text) && !utf8.RuneStart(text[end]) {
		end--
	}
	return end
}

func Store(path, text string, firstEnd int) (Body, error) {
	body := Body{size: len(text), start: firstEnd}
	if firstEnd == len(text) {
		return body, nil
	}
	if err := os.Mkdir(path, 0700); err != nil {
		return Body{}, err
	}
	body.path = path
	body.next = firstEnd
	for start := firstEnd; start < len(text); {
		end := start + End(text[start:], BlockBytes)
		file, err := os.OpenFile(body.blockPath(start), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return Body{}, errors.Join(err, body.Discard())
		}
		_, writeErr := file.WriteString(text[start:end])
		if err = errors.Join(writeErr, file.Close()); err != nil {
			return Body{}, errors.Join(err, body.Discard())
		}
		body.blocks = append(body.blocks, end)
		start = end
	}
	return body, nil
}

func (b Body) blockPath(start int) string {
	return filepath.Join(b.path, strconv.Itoa(start))
}

// Consume advances only the expected position and only after current confirms
// the caller's redaction revision. The session serializes this transaction.
func (b *Body) Consume(offset, limit int, current func() bool) (string, bool, error) {
	if len(b.blocks) == 0 || offset != b.next || limit < utf8.UTFMax {
		return "", false, nil
	}
	path := b.blockPath(b.start)
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	blockEnd := b.blocks[0]
	data := make([]byte, min(limit+1, blockEnd-offset))
	_, readErr := file.ReadAt(data, int64(offset-b.start))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return "", false, err
	}
	end := min(limit, len(data))
	for end < len(data) && end > 0 && !utf8.RuneStart(data[end]) {
		end--
	}
	if !current() {
		return "", false, nil
	}
	next := offset + end
	if next == blockEnd {
		if err := os.Remove(path); err != nil {
			return "", false, err
		}
		b.start = blockEnd
		b.blocks = b.blocks[1:]
		if next == b.size {
			b.blocks = nil
			// The session directory owner also reclaims a leftover empty directory.
			if os.Remove(b.path) == nil {
				b.path = ""
			}
			next = 0
		}
	}
	b.next = next
	return string(data[:end]), true, nil
}

func (b *Body) Discard() error {
	if b.path != "" {
		if err := os.RemoveAll(b.path); err != nil {
			return err
		}
	}
	b.path = ""
	b.blocks = nil
	b.start, b.next = b.size, 0
	return nil
}
