// Purpose: Replace one complete UTF-8 file after rechecking its original identity and revision.
// Pattern: Same-directory durable replacement.
// Usage: Reader.Save is called while the server owns a workspace effect lease.
// Related: internal/fileview/editor.go.
package fileview

import (
	"bytes"
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
	if e.redact(t.Absolute) != t.Absolute || body == nil || !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return doc, fault.New(diag.PermissionDenied, diag.UIEditorReadOnly)
	}
	original := string(body)
	if e.redact(original) != original {
		return doc, fault.New(diag.PermissionDenied, diag.UIEditorReadOnly)
	}
	if documentRevision(body) != revision {
		return doc, fault.New(diag.Conflict, diag.UIEditorChanged)
	}
	if e.redact(text) != text {
		return doc, fault.New(diag.PermissionDenied, diag.UIEditorReadOnly)
	}
	if text == original {
		return Document{Path: e.redact(t.Absolute), Revision: revision, WatchRevision: directoryRevision(info), Editable: true}, nil
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
	modeErr, syncErr := f.Chmod(info.Mode().Perm()), f.Sync()
	written, statErr := f.Stat()
	err = errors.Join(writeErr, modeErr, syncErr, statErr, f.Close())
	if err != nil {
		return doc, err
	}
	current, currentBody, currentInfo, err := e.editSource(ctx, settings, path)
	if err != nil {
		return doc, err
	}
	if current.Absolute != t.Absolute || !os.SameFile(info, currentInfo) || currentBody == nil || documentRevision(currentBody) != revision {
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
	watchRevision := ""
	// Rename changes ctime. Acknowledge the published entry, rather than the
	// temporary inode's pre-rename metadata. Concurrent external replacement
	// leaves no metadata shortcut; the next probe verifies the actual contents.
	if published, err := t.Root.Stat(t.Relative); err == nil && os.SameFile(written, published) && written.Size() == published.Size() && written.ModTime().Equal(published.ModTime()) {
		watchRevision = directoryRevision(published)
	}
	return Document{Path: e.redact(t.Absolute), Text: text, Revision: documentRevision([]byte(text)), WatchRevision: watchRevision, Editable: true}, nil
}
