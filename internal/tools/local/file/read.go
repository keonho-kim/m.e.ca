// Purpose: Read numbered UTF-8 ranges with snapshot anchors and exact byte continuation.
// Pattern: Filesystem read adapter.
// Usage: Local runtime dispatches Read.Execute after policy and target validation.
// Related: internal/tools/local/file/lines.go, internal/tools/local/dispatch.go.
package file

import (
	"context"
	"errors"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tools/local/anchors"
	resultfmt "github.com/keonho-kim/v-code/internal/tools/result"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type Read struct {
	Root           *os.Root
	Path, Absolute string
	Offset, Limit  int
	Anchors        *anchors.Store
	Redact         func(string) string
}

func (r Read) Execute(ctx context.Context) (result toolapi.Result, err error) {
	file, err := r.Root.Open(r.Path)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("only regular files can be read")
	}
	limit := r.Limit
	if limit == 0 {
		limit = 64 * 1024
	}
	first, startPartial, prefix, err := readPrefix(ctx, file, r.Offset)
	if err != nil {
		return result, err
	}
	content, err := io.ReadAll(io.LimitReader(file, int64(limit)))
	if err != nil {
		return result, err
	}
	if !utf8.Valid(content) {
		// A byte window may end inside a valid rune. Preserve a complete prefix and
		// return its exact next offset; invalid bytes and invalid starting offsets still fail.
		start := len(content) - 1
		for start > 0 && !utf8.RuneStart(content[start]) {
			start--
		}
		if start > 0 && utf8.Valid(content[:start]) && !utf8.FullRune(content[start:]) && info.Size() > int64(r.Offset+len(content)) {
			content = content[:start]
		} else {
			return result, &toolapi.Failure{Code: toolapi.InvalidArgument, Detail: "file range is not complete UTF-8 text", Next: "Use a UTF-8 boundary offset and a larger byte limit; binary files cannot be read as text."}
		}
	}
	next := r.Offset + len(content)
	truncated := info.Size() > int64(next)
	endPartial := truncated && len(content) > 0 && content[len(content)-1] != '\n'
	after, err := file.Stat()
	if err != nil {
		return result, err
	}
	if info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
		return result, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "file changed while reading", Next: "Read the range again."}
	}
	numbered, editable, err := r.numberedRead(string(content), first, startPartial, endPartial, prefix)
	if err != nil {
		return result, err
	}
	last := first + strings.Count(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(content) == 0 {
		last = 0
		first = 0
	}
	return resultfmt.Encode(struct {
		Path, Content                                 string
		Offset, NextOffset, StartLine, EndLine        int
		SizeBytes                                     int64
		Truncated, StartPartial, EndPartial, Editable bool
		Note                                          string
	}{r.Absolute, numbered, r.Offset, next, first, last, info.Size(), truncated, startPartial, endPartial, editable, "Copy line#8hex anchors from one read for edit_file. Unanchored lines are partial or redacted. Re-read after restart or anchor expiry."}, fmt.Sprintf("Read lines %d-%d (%d bytes)", first, last, len(content)))
}
