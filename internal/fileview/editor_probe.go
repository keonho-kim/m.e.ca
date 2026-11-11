// Purpose: Revalidate an editor without returning or retaining its complete file body.
// Pattern: Metadata fast path with bounded content verification when the file changed.
// Usage: File-change notifications trigger Probe; Save still checks the actual original bytes.
// Related: internal/fileview/editor.go, internal/commands/server/editor_probe.go.
package fileview

import (
	"context"
	"errors"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"os"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/policy"
)

type DocumentVersion struct {
	Revision, WatchRevision string
	Unchanged, Missing      bool
}

func (e *Reader) Probe(ctx context.Context, settings policy.Settings, path, known string) (DocumentVersion, error) {
	if len(path) > 4096 || !utf8.ValidString(path) || len(known) != 0 && len(known) != 64 {
		return DocumentVersion{}, fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
	}
	if err := ctx.Err(); err != nil {
		return DocumentVersion{}, err
	}
	if err := settings.Check(); err != nil {
		return DocumentVersion{}, err
	}
	target, err := e.resolve(settings.Scope, path)
	if errors.Is(err, os.ErrNotExist) {
		return DocumentVersion{Missing: true}, nil
	}
	if err != nil {
		return DocumentVersion{}, err
	}
	info, err := target.Root.Stat(target.Relative)
	if errors.Is(err, os.ErrNotExist) {
		return DocumentVersion{Missing: true}, nil
	}
	if err != nil {
		return DocumentVersion{}, err
	}
	if !info.Mode().IsRegular() {
		return DocumentVersion{}, fault.New(diag.InvalidArgument, diag.UIEditorReadOnly)
	}
	tag := directoryRevision(info)
	if known != "" && known == tag {
		return DocumentVersion{WatchRevision: tag, Unchanged: true}, ctx.Err()
	}
	doc, err := e.Edit(ctx, settings, path)
	if err != nil {
		return DocumentVersion{}, err
	}
	return DocumentVersion{Revision: doc.Revision, WatchRevision: doc.WatchRevision}, nil
}
