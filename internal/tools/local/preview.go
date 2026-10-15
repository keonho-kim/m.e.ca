// Purpose: Read a bounded, redacted text snapshot for user-initiated file previews.
// Pattern: Filesystem adapter.
// Usage: The file-view worker calls Preview without model tool execution or history mutation.
// Related: internal/tools/local/paths.go, internal/tui/session/file_preview.go.
package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/policy"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type FilePreview struct {
	Path, Content string
	Truncated     bool
}

func (e *Executor) Preview(ctx context.Context, settings policy.Settings, path string) (result FilePreview, err error) {
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if _, err = policy.Decide(settings, false, policy.Read); err != nil {
		return result, err
	}
	target, err := e.resolve(settings.Scope, path)
	if err != nil {
		return result, err
	}
	file, err := target.root.Open(target.relative)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("일반 파일만 미리볼 수 있습니다")
	}
	const limit = 1024 * 1024
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
		return result, fmt.Errorf("바이너리 또는 UTF-8이 아닌 파일은 텍스트 미리보기를 지원하지 않습니다")
	}
	current, err := e.resolve(settings.Scope, path)
	if err != nil {
		return result, err
	}
	after, err := current.root.Stat(current.relative)
	if err != nil {
		return result, err
	}
	if current.absolute != target.absolute || !os.SameFile(info, after) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
		return result, fmt.Errorf("파일이 변경되었습니다. 다시 여세요")
	}
	return FilePreview{Path: e.redact(target.absolute), Content: e.redact(string(body)), Truncated: truncated}, nil
}
