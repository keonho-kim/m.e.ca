// Purpose: Refresh known file observations after commands using actual scoped filesystem versions.
// Pattern: Consumer-owned filesystem observation boundary.
// Usage: Result recording refreshes facts before the next decision or completion review.
// Related: internal/agent/completion/freshness.go, internal/tools/local/versions.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/policy"
)

type fileVersions interface {
	FileVersions(context.Context, policy.Settings, []string) (map[string]string, error)
}

func (s *execution) refreshFileFacts(ctx context.Context) error {
	reader, ok := s.runner.Tools.(fileVersions)
	if !ok {
		return nil
	}
	paths := s.checkpoint.Work.FileTargets()
	if len(paths) == 0 {
		return nil
	}
	versions, err := reader.FileVersions(ctx, s.runner.Settings, paths)
	if err != nil {
		return err
	}
	s.checkpoint.Work.ReconcileFiles(versions)
	return nil
}
