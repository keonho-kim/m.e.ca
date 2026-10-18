// Purpose: Construct trusted filesystem boundaries for native launch contract tests.
// Pattern: Test fixture.
// Usage: Sandbox tests call prepareTest with temporary roots.
// Related: internal/tools/local/process/sandbox/filesystem_test.go.
package sandbox

import (
	"context"
	"github.com/keonho-kim/v-code/internal/tools/local/filesystem"
)

func prepareTest(ctx context.Context, project, home, private, command string) (Launch, error) {
	b, err := filesystem.New(project, home, private, nil)
	if err != nil {
		return Launch{}, err
	}
	return Prepare(ctx, b, command)
}
