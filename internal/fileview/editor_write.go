// Purpose: Replace one complete UTF-8 file after rechecking its original identity and revision.
// Pattern: Same-directory durable replacement.
// Usage: Reader.Save is called while the server owns a workspace effect lease.
// Related: internal/fileview/editor.go.
package fileview

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

func (e *Reader) Save(ctx context.Context, settings policy.Settings, path, text, revision string) (doc Document, resultErr error) {
	if len(text) > EditorBytes || !utf8.ValidString(text) || strings.IndexByte(text, 0) >= 0 || len(revision) != 64 {
		return doc, fault.New(diag.InvalidArgument, diag.UIEditorInvalid)
	}
	t, body, info, err := e.editSource(ctx, settings, path)
	if err != nil {
		return doc, err
	}
	if e.redact(t.Absolute) != t.Absolute || len(body) > EditorBytes || !utf8.Valid(body) || strings.IndexByte(string(body), 0) >= 0 || e.redact(string(body)) != string(body) {
		return doc, fault.New(diag.PermissionDenied, diag.UIEditorReadOnly)
	}
	if documentRevision(body) != revision {
		return doc, fault.New(diag.Conflict, diag.UIEditorChanged)
	}
	if e.redact(text) != text {
		return doc, fault.New(diag.PermissionDenied, diag.UIEditorReadOnly)
	}
	if text == string(body) {
		return Document{Path: e.redact(t.Absolute), Revision: revision, Editable: true}, nil
	}
	tmp := filepath.Join(filepath.Dir(t.Relative), ".meca-edit-"+rand.Text())
	f, err := t.Root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return doc, err
	}
	defer func() {
		if err := t.Root.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	_, writeErr := f.WriteString(text)
	err = errors.Join(writeErr, f.Chmod(info.Mode().Perm()), f.Sync(), f.Close())
	if err != nil {
		return doc, err
	}
	current, currentBody, currentInfo, err := e.editSource(ctx, settings, path)
	if err != nil {
		return doc, err
	}
	if current.Absolute != t.Absolute || !os.SameFile(info, currentInfo) || documentRevision(currentBody) != revision {
		return doc, fault.New(diag.Conflict, diag.UIEditorChanged)
	}
	if err := ctx.Err(); err != nil {
		return doc, err
	}
	if err := t.Root.Rename(tmp, t.Relative); err != nil {
		return doc, err
	}
	dir, err := t.Root.Open(filepath.Dir(t.Relative))
	if err != nil {
		return doc, err
	}
	if err := errors.Join(dir.Sync(), dir.Close()); err != nil {
		return doc, err
	}
	return Document{Path: e.redact(t.Absolute), Text: text, Revision: documentRevision([]byte(text)), Editable: true}, nil
}
