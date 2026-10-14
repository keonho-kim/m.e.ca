// Purpose: Number read ranges and anchor only complete unredacted lines.
// Pattern: Read projection.
// Usage: Read.Execute calls numberedRead with its observed prefix and partial-line flags.
// Related: internal/tools/local/file/read.go, internal/tools/local/anchors/store.go.
package file

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"strings"
)

// readPrefix counts lines without retaining the file prefix in memory.
func readPrefix(ctx context.Context, reader io.Reader, offset int) (line int, partial bool, prefix hash.Hash, err error) {
	line = 1
	prefix = sha256.New()
	buffer := make([]byte, 32*1024)
	for offset > 0 {
		if err := ctx.Err(); err != nil {
			return 0, false, nil, err
		}
		n, err := io.ReadFull(reader, buffer[:min(offset, len(buffer))])
		if n > 0 {
			if _, err := prefix.Write(buffer[:n]); err != nil {
				return 0, false, nil, err
			}
			line += bytes.Count(buffer[:n], []byte{'\n'})
			partial = buffer[n-1] != '\n'
			offset -= n
		}
		if err != nil {
			return 0, false, nil, fmt.Errorf("offset exceeds file length: %w", err)
		}
	}
	return line, partial, prefix, nil
}
func (r Read) numberedRead(content string, first int, startPartial, endPartial bool, prefix hash.Hash) (string, bool, error) {
	display := strings.ReplaceAll(content, "\r\n", "\n")
	redacted := r.Redact(display)
	pieces := strings.SplitAfter(content, "\n")
	if pieces[len(pieces)-1] == "" {
		pieces = pieces[:len(pieces)-1]
	}
	if len(pieces) == 0 {
		return "", false, nil
	}
	begin, finish := 0, len(pieces)
	if startPartial {
		begin = 1
		if _, err := prefix.Write([]byte(pieces[0])); err != nil {
			return "", false, err
		}
	}
	if endPartial {
		finish--
	}
	if redacted != display {
		begin = finish
		if strings.Count(redacted, "\n") != strings.Count(content, "\n") {
			return fmt.Sprintf("%d-%d | [redacted: line mapping cannot be preserved]\n", first, first+len(pieces)-1), false, nil
		}
		pieces = strings.SplitAfter(redacted, "\n")
		if pieces[len(pieces)-1] == "" {
			pieces = pieces[:len(pieces)-1]
		}
	}
	var out strings.Builder
	for i := 0; i < len(pieces); {
		if i == begin && begin < finish {
			var digest [32]byte
			copy(digest[:], prefix.Sum(nil))
			anchored, err := r.Anchors.Record(r.Absolute, first+i, digest, strings.Join(pieces[begin:finish], ""))
			if err != nil {
				return "", false, err
			}
			out.WriteString(strings.ReplaceAll(anchored, "\r\n", "\n"))
			i = finish
			continue
		}
		fmt.Fprintf(&out, "%d | %s", first+i, strings.ReplaceAll(pieces[i], "\r\n", "\n"))
		if !strings.HasSuffix(pieces[i], "\n") {
			out.WriteByte('\n')
		}
		i++
	}
	return out.String(), begin < finish, nil
}
