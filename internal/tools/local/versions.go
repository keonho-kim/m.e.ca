// Purpose: Report stable versions for known targets without executing model-generated commands.
// Pattern: Scoped filesystem observation adapter.
// Usage: The run owner refreshes known file facts through Executor.FileVersions.
// Related: internal/tools/local/version.go, internal/agent/run/file_facts.go.
package local

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (e *Executor) FileVersions(ctx context.Context, settings policy.Settings, paths []string) (map[string]string, error) {
	if _, err := policy.Decide(settings, policy.Read); err != nil {
		return nil, err
	}
	versions := make(map[string]string, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		target, err := e.resolve(settings.Scope, path)
		if err != nil {
			continue
		}
		value, err := snapshot(target)
		if err != nil {
			continue
		}
		versions[path] = versionValue(value)
	}
	return versions, nil
}
func versionValue(v version) string {
	if !v.exists {
		return "missing"
	}
	if v.info.IsDir() {
		return "directory"
	}
	return fmt.Sprintf("%x", v.digest)
}
