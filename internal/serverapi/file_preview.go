// Purpose: Page immutable redacted file previews without rereading the full file for every page.
// Pattern: Revision-bound workspace read through the existing bounded chunk cache.
// Usage: previewFile requires the returned revision on subsequent pages and validates access each time.
// Related: internal/commands/server/workspace.go, internal/serverapi/detail/pool.go.
package serverapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/detail"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"strings"
	"unicode/utf8"
)

func (a *API) previewFile(ctx context.Context, r *v1.FileRequest) (*v1.FilePreview, error) {
	offset, limit := int64(r.GetOffset()), int(r.GetLimit())
	if limit == 0 {
		limit = 64 << 10
	}
	if offset < 0 || limit < 4 || limit > 64<<10 || offset > 0 && r.GetRevision() == "" || len(r.GetRevision()) > 129 {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	path, err := a.Service.FileReference(ctx, r.GetPath())
	if err != nil {
		return nil, err
	}
	revision, expected := "", ""
	if r.GetRevision() != "" {
		var ok bool
		revision, expected, ok = strings.Cut(r.GetRevision(), ".")
		if !ok || len(revision) != 64 || len(expected) != 64 {
			return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
		}
	}
	var text *string
	truncated := len(revision) > 0 && revision[0] == 't'
	load := func() error {
		value, err := a.Service.Preview(ctx, r.GetPath())
		if err != nil {
			return err
		}
		digest := sha256.Sum256([]byte(value.Content))
		current := hex.EncodeToString(digest[:])
		if value.Truncated {
			current = "t" + current[1:]
		} else {
			current = "f" + current[1:]
		}
		if revision != "" && current != revision {
			return faults.New(diagcatalog.Conflict, diagcatalog.ServerPageInvalid)
		}
		revision = current
		text = &value.Content
		truncated = value.Truncated
		return nil
	}
	if revision == "" {
		if err := load(); err != nil {
			return nil, err
		}
	}
	page, total, token, err := a.Details.Read(ctx, detail.Request{Expected: expected, Session: "workspace", File: path, FileRevision: revision, Version: a.RedactionVersion, NewMask: a.NewMask, LoadText: func(reserve func(int) error) (string, error) {
		if text == nil {
			if err := load(); err != nil {
				return "", err
			}
		}
		if err := reserve(len(*text)); err != nil {
			return "", err
		}
		return *text, nil
	}}, offset, limit)
	if err != nil {
		return nil, err
	}
	if len(page) > 0 && !utf8.RuneStart(page[0]) {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	for len(page) > 0 && !utf8.Valid(page) {
		page = page[:len(page)-1]
	}
	return &v1.FilePreview{Path: path, Text: string(page), Offset: offset, NextOffset: offset + int64(len(page)), Truncated: truncated || offset+int64(len(page)) < total, Revision: revision + "." + token}, nil
}
