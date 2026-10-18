// Purpose: Construct trusted boundaries for process integration fixtures.
// Pattern: Test fixture.
// Usage: Process tests invoke runTest with temporary project roots.
// Related: internal/tools/local/process/run_unix.go.
package process

import (
	"context"
	"github.com/keonho-kim/v-code/internal/tools/local/filesystem"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func runTest(ctx context.Context, project, home, private, command string) (Output, error) {
	if project == home {
		home = filepath.Dir(project)
	}
	b, err := filesystem.New(project, home, private, nil)
	if err != nil {
		return Output{Status: "scope_unavailable", ExitCode: -1}, err
	}
	return Run(ctx, b, command)
}

func requireTestBoundary(t *testing.T, b *filesystem.Boundary) {
	t.Helper()
	output, err := Run(context.Background(), b, "true")
	if err != nil {
		if runtime.GOOS == "linux" && output.Status == "scope_unavailable" && os.Getenv("V_SANDBOX_REQUIRED") != "1" {
			t.Skipf("native kernel containment unavailable: %v", err)
		}
		t.Fatal(err)
	}
}
