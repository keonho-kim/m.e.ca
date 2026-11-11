// Purpose: Resolve immutable effect scopes for known file operations before lock admission.
// Pattern: Filesystem boundary parser.
// Usage: Server rechecks these canonical paths after acquiring its shared effect gate.
// Related: internal/commands/server/file_effect.go, internal/fileview/reader.go.
package fileview

import (
	"context"

	"github.com/keonho-kim/meca/internal/policy"
)

func (e *Reader) EffectPaths(ctx context.Context, settings policy.Settings, paths ...string) ([]string, error) {
	if err := settings.Check(); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		target, err := e.resolve(settings.Scope, path)
		if err != nil {
			return nil, err
		}
		out = append(out, target.Absolute)
	}
	return out, nil
}
