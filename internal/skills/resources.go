// Purpose: Page UTF-8 references while keeping file reads inside a verified skill directory.
// Pattern: Confined resource reader.
// Usage: read_skill_file and authenticated server resource previews call ReadFile.
// Related: internal/skills/files.go, internal/skills/reader.go.
package skills

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/redaction"
)

type Resource struct {
	Revision   string `json:"revision"`
	Text       string `json:"text,omitempty"`
	MediaType  string `json:"media_type"`
	Offset     int64  `json:"offset"`
	NextOffset int64  `json:"next_offset"`
	TotalBytes int64  `json:"total_bytes"`
	Binary     bool   `json:"binary,omitempty"`
}

func (r *Reader) ReadFile(ctx context.Context, id, path string, offset int64, limit int, revision string) (value Resource, err error) {
	if offset < 0 || limit < 1 || limit > MaxResourceBytes {
		return value, ErrInvalid
	}
	lease, err := r.OpenFile(ctx, id, path)
	if err != nil {
		return value, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	return lease.ReadPage(ctx, offset, limit, revision)
}

func (f *ResourceFile) ReadPage(ctx context.Context, offset int64, limit int, revision string) (value Resource, err error) {
	if offset < 0 || limit < 1 || limit > MaxResourceBytes {
		return value, ErrInvalid
	}
	if err := f.check(ctx); err != nil {
		return value, err
	}
	file, info := f.file, f.info

	if offset > 0 && revision == "" {
		return value, faults.Errorf(diagcatalog.SkillsResourceRevisionRequired, ErrInvalid)
	}
	if revision != "" && revision != f.Revision {
		return value, faults.Errorf(diagcatalog.SkillsResourceRevisionChanged, ErrInvalid)
	}
	value.Revision = f.Revision
	value.TotalBytes, value.Offset = info.Size(), offset
	if offset > info.Size() {
		return value, ErrInvalid
	}
	probe := make([]byte, 512)
	n, readErr := file.ReadAt(probe, 0)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return value, readErr
	}
	value.MediaType = http.DetectContentType(probe[:n])
	if strings.IndexByte(string(probe[:n]), 0) >= 0 || !utf8.Valid(probe[:n]) && n < len(probe) || strings.HasPrefix(value.MediaType, "image/") {
		value.Binary = true
		return value, f.check(ctx)
	}
	windowStart := max(int64(0), offset-redaction.MaxCredentialBytes)
	windowEnd := min(info.Size(), offset+int64(limit)+utf8.UTFMax+redaction.MaxCredentialBytes)
	buffer := make([]byte, windowEnd-windowStart)
	n, readErr = file.ReadAt(buffer, windowStart)
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return value, readErr
	}
	if n != len(buffer) {
		return value, ErrInvalid
	}
	start := int(offset - windowStart)
	end := min(start+limit, len(buffer))
	selected := buffer[start:end]
	if len(selected) > 0 && !utf8.RuneStart(selected[0]) {
		return value, ErrInvalid
	}
	if bytes.IndexByte(selected, 0) >= 0 {
		value.MediaType, value.Binary = "application/octet-stream", true
		return value, f.check(ctx)
	}
	for len(selected) > 0 && !utf8.Valid(selected) {
		if len(selected) <= limit-utf8.UTFMax {
			value.MediaType, value.Binary = "application/octet-stream", true
			return value, f.check(ctx)
		}
		selected = selected[:len(selected)-1]
	}
	if len(selected) == 0 && offset < info.Size() {
		return value, ErrInvalid
	}
	value.Text, err = f.reader.cleanRange(string(buffer), start, start+len(selected))
	if err != nil {
		return Resource{}, err
	}
	if len(value.Text) > MaxResourceBytes {
		return Resource{}, ErrLimit
	}
	if offset+int64(len(selected)) < info.Size() {
		value.NextOffset = offset + int64(len(selected))
	}
	if err := f.check(ctx); err != nil {
		return Resource{}, err
	}
	return value, nil
}
