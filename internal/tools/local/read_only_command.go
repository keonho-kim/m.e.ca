// Purpose: Bind validated LAT queries to fixed host executables and an inert environment.
// Pattern: Read-only command execution boundary.
// Usage: A one-shot LAT operation revalidates immediately before the native process start.
// Related: internal/tools/local/read_only.go, internal/policy/readonly/parse.go.
package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/policy/readonly"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

func (op *readOnlyOperation) executeQuery(ctx context.Context) (toolapi.Result, error) {
	if err := op.command.settings.Check(); err != nil {
		return toolapi.Result{}, err
	}
	if op.command.settings.Scope != policy.System {
		return toolapi.Result{}, fmt.Errorf("LAT host queries require system Scope; directory confinement is unavailable")
	}
	if textformat.RedactSource(op.command.input.Command, op.executor.redact) != op.command.input.Command {
		return toolapi.Result{}, fmt.Errorf("tool arguments contain a protected credential")
	}
	if err := op.executor.boundary.Check(); err != nil {
		return toolapi.Result{}, err
	}
	query, err := readonly.Parse(op.command.input.Command)
	if err != nil {
		return toolapi.Result{}, err
	}
	command, err := query.Render(readOnlyExecutable)
	if err != nil {
		return toolapi.Result{}, err
	}
	// Start only this canonical literal query, never the original shell text.
	commandOp := &operation{executor: op.command.executor, settings: op.command.settings, input: op.command.input, callID: op.command.callID}
	commandOp.input.Command = command
	commandOp.environment = []string{
		"PATH=/usr/bin:/bin", "LC_ALL=C", "LANG=C", "HOME=" + op.executor.artifacts,
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_COUNT=0", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat", "PAGER=cat", "GIT_ATTR_NOSYSTEM=1", "GIT_NO_REPLACE_OBJECTS=1", "GIT_NO_LAZY_FETCH=1",
		"GIT_ALLOW_PROTOCOL=", "GIT_PROTOCOL_FROM_USER=0",
	}
	return commandOp.command(ctx)
}

func readOnlyExecutable(name string) (string, error) {
	// Never consult inherited PATH, cwd, shell functions or project-local tools.
	// Host installations in these fixed locations are trusted application inputs.
	directories := []string{"/usr/bin", "/bin"}
	trustedRoots := []string{"/usr/bin", "/bin"}
	if name == "rg" {
		directories = append(directories, "/opt/homebrew/bin", "/usr/local/bin")
		trustedRoots = append(trustedRoots, "/opt/homebrew", "/usr/local")
	}
	return lookupReadOnlyExecutable(name, directories, trustedRoots)
}

func lookupReadOnlyExecutable(name string, directories, trustedRoots []string) (string, error) {
	for _, directory := range directories {
		path := filepath.Join(directory, name)
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		info, err := os.Stat(resolved)
		trusted := false
		for _, root := range trustedRoots {
			relative, relErr := filepath.Rel(root, resolved)
			if relErr == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				trusted = true
				break
			}
		}
		if err == nil && trusted && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 && info.Mode().Perm()&0022 == 0 {
			// Multicall binaries select behavior using argv[0]. Validate the
			// canonical target but retain the command's original absolute alias.
			return path, nil
		}
	}
	return "", fmt.Errorf("LAT read-only query: trusted host executable %q is unavailable in fixed system locations; use another supported read/search command", name)
}
