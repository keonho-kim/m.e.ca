// Purpose: Borrow current workspace policy for lightweight editor revalidation.
// Pattern: Consumer-owned filesystem read boundary.
// Usage: Native editorProbe separates dirty hints from actual content-version changes.
// Related: internal/fileview/editor_probe.go, internal/serverapi/application.go.
package server

import (
	"context"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
)

type editorProbe interface {
	Probe(context.Context, policy.Settings, string, string) (fileview.DocumentVersion, error)
}

func (s *Service) EditorProbe(ctx context.Context, path, known string) (fileview.DocumentVersion, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return fileview.DocumentVersion{}, err
	}
	defer done()
	files, settings, err := s.fileSettings()
	if err != nil {
		return fileview.DocumentVersion{}, err
	}
	reader, ok := files.(editorProbe)
	if !ok {
		return fileview.DocumentVersion{}, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	return reader.Probe(ctx, settings, path, known)
}
