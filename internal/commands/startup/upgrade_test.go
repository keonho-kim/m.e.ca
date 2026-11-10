// Purpose: Verify explicit upgrade distribution selection without requiring a project directory.
// Pattern: Startup contract tests.
// Usage: go test ./internal/commands/startup -run TestUpgradeDistributionFlags.
// Related: internal/commands/startup/parse.go, internal/distribution/variants.go.
package startup

import (
	"testing"

	"github.com/keonho-kim/meca/internal/distribution"
)

func TestUpgradeDistributionFlags(t *testing.T) {
	for _, tc := range []struct {
		args []string
		kind distribution.Kind
	}{
		{[]string{"upgrade"}, distribution.Name},
		{[]string{"upgrade", "--tui"}, distribution.TUI},
		{[]string{"upgrade", "--full"}, distribution.Full},
	} {
		mode, err := Parse(tc.args, "/missing/project")
		if err != nil || mode.Action != Upgrade || mode.Project != "" || mode.UpgradeKind != tc.kind {
			t.Errorf("upgrade arguments %v: %+v, %v", tc.args, mode, err)
		}
	}
	for _, args := range [][]string{
		{"upgrade", "--tui", "--full"}, {"upgrade", "--full", "--tui"},
		{"upgrade", "--tui", "--tui"}, {"upgrade", "--full", "--full"},
		{"upgrade", "--minimum"}, {"upgrade", "--tui=false"},
		{"upgrade", "--full=false"}, {"upgrade", "--tui", "extra"},
	} {
		if _, err := Parse(args, "/missing/project"); err == nil {
			t.Errorf("unsupported upgrade arguments accepted: %v", args)
		}
	}
}
