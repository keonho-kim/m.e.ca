// Purpose: Serialize explicit folder creation with running agent and editor file effects.
// Pattern: Application operation over a consumer-owned filesystem contract.
// Usage: Native createFolder requests call CreateFolder in the selected workspace runtime.
// Related: internal/fileview/directory_create.go, internal/serverapi/application.go.
package server

import (
	"context"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

type folderCreator interface {
	CreateDirectory(context.Context, policy.Settings, string, string) (string, error)
}

func (s *Service) CreateFolder(ctx context.Context, parent, name string) (string, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return "", err
	}
	defer done()
	s.mu.Lock()
	settings := policy.Settings{Scope: policy.System, Permission: s.currentConfig().Permission}
	s.mu.Unlock()
	files, ok := s.options.Files.(folderCreator)
	if !ok {
		return "", fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	ctx, release, err := s.beginFileEffect(ctx, settings, parent)
	if err != nil {
		return "", err
	}
	defer release()
	return files.CreateDirectory(ctx, settings, parent, name)
}
