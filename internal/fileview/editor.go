// Purpose: Read an editor-sized file with stable identity and classify oversized files from metadata.
// Pattern: Guarded filesystem adapter.
// Usage: Native editor operations borrow Reader roots under the server's workspace effect lease.
// Related: internal/fileview/reader.go, internal/fileview/editor_write.go.
package fileview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"syscall"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

// Complete documents remain small enough for syntax parsing and every transport frame.
// Larger files retain paged preview access.
const EditorBytes = 2 << 20

type Document struct {
	Path, Text, Revision, WatchRevision, Reason string
	Editable                                    bool
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
	var body []byte
	if info.Size() >= 0 && info.Size() <= EditorBytes {
		// One extra byte distinguishes a concurrent extension from the recorded size.
		body = make([]byte, int(info.Size())+1)
		var n int
		n, err = io.ReadFull(f, body)
		body = body[:n]
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			err = nil
		}
	}
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
	if current.Absolute != t.Absolute || !os.SameFile(info, entry) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) || !info.ModTime().Equal(entry.ModTime()) || info.Size() != entry.Size() || (body != nil && int64(len(body)) != info.Size()) {
		return t, nil, nil, fault.New(diag.Conflict, diag.UIEditorChanged)
	}
	return t, body, info, nil
}

func documentRevision(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (e *Reader) Edit(ctx context.Context, settings policy.Settings, path string) (Document, error) {
	t, body, info, err := e.editSource(ctx, settings, path)
	if err != nil {
		return Document{}, err
	}
	doc := Document{Path: e.redact(t.Absolute), WatchRevision: directoryRevision(info)}
	if body == nil || !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		doc.Reason = diag.Text(diag.UIEditorReadOnly)
		return doc, nil
	}
	text := string(body)
	doc.Text = e.redact(text)
	doc.Editable = doc.Text == text && doc.Path == t.Absolute
	if !doc.Editable {
		doc.Reason = diag.Text(diag.UIEditorReadOnly)
		return doc, nil
	}
	doc.Revision = documentRevision(body)
	return doc, nil
}
