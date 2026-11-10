// Purpose: Read complete editable text and commit revision-checked file replacements.
// Pattern: Guarded filesystem adapter.
// Usage: Native editor operations borrow Reader roots under the server's workspace effect lease.
// Related: internal/fileview/reader.go, internal/fileview/editor_write.go.
package fileview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

// Complete documents remain small enough for syntax parsing and every transport frame.
// Larger files retain paged preview access.
const EditorBytes = 1 << 20

type Document struct {
	Path, Text, Revision, Reason string
	Editable                     bool
}

func (e *Reader) editSource(ctx context.Context, settings policy.Settings, path string) (Target, []byte, os.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return Target{}, nil, nil, err
	}
	if err := settings.Check(); err != nil {
		return Target{}, nil, nil, err
	}
	t, err := e.resolve(settings.Scope, path)
	if err != nil {
		return t, nil, nil, err
	}
	f, err := t.Root.OpenFile(t.Relative, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return t, nil, nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return t, nil, nil, errors.Join(err, f.Close(), fault.New(diag.InvalidArgument, diag.UIEditorReadOnly))
	}
	body, err := io.ReadAll(io.LimitReader(f, EditorBytes+1))
	after, statErr := f.Stat()
	err = errors.Join(err, statErr, f.Close(), ctx.Err())
	if err != nil {
		return t, nil, nil, err
	}
	current, err := e.resolve(settings.Scope, path)
	if err != nil {
		return t, nil, nil, err
	}
	entry, err := current.Root.Stat(current.Relative)
	if err != nil {
		return t, nil, nil, err
	}
	if current.Absolute != t.Absolute || !os.SameFile(info, entry) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) || !info.ModTime().Equal(entry.ModTime()) || info.Size() != entry.Size() {
		return t, nil, nil, fault.New(diag.Conflict, diag.UIEditorChanged)
	}
	return t, body, info, nil
}

func documentRevision(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (e *Reader) Edit(ctx context.Context, settings policy.Settings, path string) (Document, error) {
	t, body, _, err := e.editSource(ctx, settings, path)
	if err != nil {
		return Document{}, err
	}
	doc := Document{Path: e.redact(t.Absolute)}
	if len(body) > EditorBytes || !utf8.Valid(body) || strings.IndexByte(string(body), 0) >= 0 {
		doc.Reason = diag.Text(diag.UIEditorReadOnly)
		return doc, nil
	}
	doc.Text = e.redact(string(body))
	doc.Editable = doc.Text == string(body) && doc.Path == t.Absolute
	if !doc.Editable {
		doc.Reason = diag.Text(diag.UIEditorReadOnly)
		return doc, nil
	}
	doc.Revision = documentRevision(body)
	return doc, nil
}
