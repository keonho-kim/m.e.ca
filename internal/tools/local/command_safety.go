// Purpose: Bind shell safety analysis to canonical filesystem observations and a frozen environment.
// Pattern: Filesystem policy adapter.
// Usage: Command preparation and execution compare the same analysis inputs and target identities.
// Related: internal/policy/command/analyze.go, internal/tools/local/prepare.go, internal/tools/local/operation.go.
package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	commandpolicy "github.com/keonho-kim/v-code/internal/policy/command"
)

type shellTarget struct {
	path string
	info os.FileInfo
}
type shellSafety struct {
	decision policy.Safety
	findings []commandpolicy.Finding
	targets  []shellTarget
}

func (e *Executor) analyzeCommand(ctx context.Context, text string, env []string) (shellSafety, error) {
	var result shellSafety
	if err := e.boundary.Check(); err != nil {
		return result, err
	}
	report, err := commandpolicy.Analyze(ctx, text, commandpolicy.Context{
		Project: e.project, Home: e.home, Environment: env,
		Resolve: func(directory, path string, removal bool) (commandpolicy.ResolvedPath, error) {
			if !filepath.IsAbs(path) {
				path = directory + "/" + path
			}
			// unlink removes the final symlink itself, while a trailing slash
			// or an intermediate symlink refers to its destination directory.
			resolved, err := shellPath(path, removal)
			if err != nil {
				return commandpolicy.ResolvedPath{}, err
			}
			info, err := os.Lstat(resolved)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return commandpolicy.ResolvedPath{}, err
			}
			result.targets = append(result.targets, shellTarget{resolved, info})
			return commandpolicy.ResolvedPath{Path: resolved, Directory: info != nil && info.IsDir(), Symlink: info != nil && info.Mode()&os.ModeSymlink != 0}, nil
		},
	})
	result.decision, result.findings = report.Safety, report.Findings
	var invalid *commandpolicy.SyntaxError
	var blocked *commandpolicy.BlockedError
	if errors.As(err, &invalid) {
		err = &toolapi.Failure{Code: toolapi.InvalidCommand, Detail: err.Error(), Next: "Correct the command syntax before retrying.", Cause: err}
	}
	if errors.As(err, &blocked) {
		err = &toolapi.Failure{Code: toolapi.CommandBlocked, Detail: err.Error(), Next: "Respect the protected target or privilege restriction; do not retry the same effect through another command.", Cause: err}
	}
	return result, err
}

func (s shellSafety) same(other shellSafety) bool {
	if !reflect.DeepEqual(s.decision, other.decision) || !reflect.DeepEqual(s.findings, other.findings) || len(s.targets) != len(other.targets) {
		return false
	}
	for i, target := range s.targets {
		current := other.targets[i]
		if target.path != current.path || (target.info == nil) != (current.info == nil) {
			return false
		}
		if target.info != nil && !os.SameFile(target.info, current.info) {
			return false
		}
	}
	return true
}
