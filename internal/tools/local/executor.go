// Purpose: Own project and host filesystem handles and captured-output cleanup for local tools.
// Pattern: Filesystem Adapter lifecycle.
// Usage: Composition root creates and closes Executor.
// Related: internal/tools/local/prepare.go, internal/tools/local/paths.go.
package local

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	"os"
	"path/filepath"
	"time"
)

type Executor struct {
	imageReads              chan struct{}
	liveOutputs             liveCaptures
	Files                   *fileview.Reader
	commands                process.Registry
	commandWait             time.Duration
	project, home           string
	artifacts               string
	artifactLease           *os.File
	projectRoot, systemRoot *os.Root
	redact                  func(string) string
	boundary                *filesystem.Boundary
}

func NewExecutor(project, home, private string, redact func(string) string) (*Executor, error) {
	if redact == nil {
		return nil, fmt.Errorf("a secret redactor is required")
	}
	boundary, err := filesystem.New(project, home, private)
	if err != nil {
		return nil, err
	}
	project, home = boundary.Project(), boundary.Home()
	root, err := os.OpenRoot(project)
	if err != nil {
		return nil, err
	}
	system, err := os.OpenRoot(filepath.VolumeName(project) + string(filepath.Separator))
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	executor := &Executor{imageReads: make(chan struct{}, 1), project: project, home: home, projectRoot: root, systemRoot: system, redact: redact, boundary: boundary, commandWait: process.InitialWait}
	executor.Files = fileview.New(project, func(scope policy.Scope, path string) (fileview.Target, error) {
		target, err := executor.resolve(scope, path)
		return fileview.Target{Absolute: target.absolute, Relative: target.relative, Root: target.root}, err
	}, func(text string) string { return executor.redact(text) })
	executor.commands.Save = executor.archiveCommand
	executor.artifacts, executor.artifactLease, err = newArtifacts()
	if err != nil {
		return nil, errors.Join(err, executor.Close())
	}
	return executor, nil
}

func (executor *Executor) Close() error {
	executor.commands.End()
	executor.liveOutputs.clear()
	err := errors.Join(executor.projectRoot.Close(), executor.systemRoot.Close())
	if executor.artifacts != "" {
		err = errors.Join(err, os.RemoveAll(executor.artifacts))
	}
	if executor.artifactLease != nil {
		err = errors.Join(err, executor.artifactLease.Close())
		executor.artifactLease = nil
	}
	return err
}
