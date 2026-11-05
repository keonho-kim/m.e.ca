// Purpose: Build isolated environment and filesystem inputs for static shell policy tests.
// Pattern: Test-only factory for immutable prepared programs.
// Usage: Analyze fixtures parse fresh programs; reuse tests call Program.Analyze directly.
// Related: internal/policy/command/analyze_test.go, internal/policy/command/program.go.
package command

import (
	"context"
	"path/filepath"
)

type analysisInput struct {
	Context
	Environment []string
}

func testContext() analysisInput {
	return analysisInput{
		Environment: []string{"HOME=/home/user", "PATH=/usr/bin:/bin"},
		Context: Context{Project: "/work/project", Home: "/home/user", Resolve: func(dir, path string, _ bool) (ResolvedPath, error) {
			if !filepath.IsAbs(path) {
				path = filepath.Join(dir, path)
			}
			path = filepath.Clean(path)
			directory := path == "/work" || path == "/work/project" || path == "/work/project/out" || path == "/etc"
			return ResolvedPath{Path: path, Directory: directory}, nil
		}},
	}
}

func Analyze(ctx context.Context, text string, input analysisInput) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	program, err := Parse(text, input.Environment)
	if err != nil {
		return Report{}, err
	}
	return program.Analyze(ctx, input.Context)
}
