// Purpose: Read a bounded, redacted text snapshot for user-initiated file previews.
// Pattern: Filesystem adapter.
// Usage: The file-view worker calls Preview without model tool execution or history mutation.
// Related: internal/fileview/reader.go, internal/tui/session/file_preview.go.
package fileview

import (
	"context"
	"errors"

	"io"
	"os"
	"strings"
	"syscall"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

type Preview struct {
	Path, Content string
	Truncated     bool
}

func (e *Reader) Preview(ctx context.Context, settings policy.Settings, path string) (result Preview, err error) {
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if err = settings.Check(); err != nil {
		return result, err
	}
	target, err := e.resolve(settings.Scope, path)
	if err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	file, err := target.Root.OpenFile(target.Relative, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, faults.Errorf(diagcatalog.FileviewWrappedCause)
	}
	const limit = 16 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	truncated := len(body) > limit
	if truncated {
		body = body[:limit]
		for n := 0; n < utf8.UTFMax && !utf8.Valid(body); n++ {
			body = body[:len(body)-1]
		}
	}
	if !utf8.Valid(body) || strings.IndexByte(string(body), 0) >= 0 {
		return result, faults.Errorf(diagcatalog.FileviewUTF)
	}
	current, err := e.resolve(settings.Scope, path)
	if err != nil {
		return result, err
	}
	after, err := current.Root.Stat(current.Relative)
	if err != nil {
		return result, err
	}
	if current.Absolute != target.Absolute || !os.SameFile(info, after) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
		return result, faults.Errorf(diagcatalog.FileviewWrappedCause2)
	}
	return Preview{Path: e.redact(target.Absolute), Content: e.redact(string(body)), Truncated: truncated}, nil
}
