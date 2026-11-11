// Purpose: Reserve canonical file scopes and reject aliases that change during gate admission.
// Pattern: Application-owned cancellable effect lease.
// Usage: Editor, folder creation and entry mutation share the process-wide gate.
// Related: internal/fileview/effect_paths.go, internal/tools/workspacegate/gate.go.
package server

import (
	"context"
	"slices"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/workspacegate"
)

type fileEffectPaths interface {
	EffectPaths(context.Context, policy.Settings, ...string) ([]string, error)
}

func (s *Service) beginFileEffect(ctx context.Context, settings policy.Settings, paths ...string) (context.Context, func(), error) {
	resolver, scoped := s.options.Files.(fileEffectPaths)
	var resolved []string
	var err error
	if scoped {
		resolved, err = resolver.EffectPaths(ctx, settings, paths...)
		if err != nil {
			return nil, nil, err
		}
	}
	gate := s.options.Effects
	if gate == nil {
		gate = &s.workspace
	}
	ctx, end := workspacegate.BindPaths(ctx, gate, resolved)
	release, err := workspacegate.Acquire(ctx)
	if err != nil {
		end()
		return nil, nil, err
	}
	done := func() { release(); end() }
	if scoped {
		current, err := resolver.EffectPaths(ctx, settings, paths...)
		if err != nil {
			done()
			return nil, nil, err
		}
		if !slices.Equal(current, resolved) {
			done()
			return nil, nil, fault.New(diag.Conflict, diag.UIFileMutationChanged)
		}
	}
	return ctx, done, nil
}
