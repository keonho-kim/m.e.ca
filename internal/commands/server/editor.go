// Purpose: Apply current workspace policy and serialize interactive file edits with agent effects.
// Pattern: Application boundary around a consumer-owned editor contract.
// Usage: Native editor commands and file observers borrow the workspace's read-only source.
// Related: internal/fileview/editor.go, internal/tools/workspacegate/gate.go.
package server

import (
	"context"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/workspacegate"
)

type editor interface {
	Edit(context.Context, policy.Settings, string) (fileview.Document, error)
	Save(context.Context, policy.Settings, string, string, string) (fileview.Document, error)
}

func (s *Service) Editor(ctx context.Context, path, text, revision string, save bool) (fileview.Document, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return fileview.Document{}, err
	}
	defer done()
	s.mu.Lock()
	cfg := s.currentConfig()
	s.mu.Unlock()
	settings := policy.Settings{Scope: policy.System, Permission: cfg.Permission}
	err = settings.Check()
	if err != nil {
		return fileview.Document{}, err
	}
	files, ok := s.options.Files.(editor)
	if !ok {
		return fileview.Document{}, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	if !save {
		return files.Edit(ctx, settings, path)
	}
	gate := s.options.Effects
	if gate == nil {
		gate = &s.workspace
	}
	ctx, end := workspacegate.Bind(ctx, gate)
	defer end()
	release, err := workspacegate.Acquire(ctx)
	if err != nil {
		return fileview.Document{}, err
	}
	defer release()
	return files.Save(ctx, settings, path, text, revision)
}
