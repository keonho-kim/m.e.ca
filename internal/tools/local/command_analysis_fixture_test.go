// Purpose: Supply fresh-parse command analysis to filesystem policy fixtures.
// Pattern: Test-only adapter over the prepared program analysis path.
// Usage: Local policy tests request analysis without starting a process or retaining a candidate.
// Related: internal/tools/local/command_safety.go, internal/tools/local/command_binding_test.go.
package local

import (
	"context"

	commandpolicy "github.com/keonho-kim/meca/internal/policy/command"
)

func (e *Executor) analyzeCommand(ctx context.Context, text string, env []string) (shellSafety, error) {
	program, err := commandpolicy.Parse(text, env)
	if err != nil {
		return shellSafety{}, commandAnalysisError(err)
	}
	return e.analyzeProgram(ctx, program)
}
