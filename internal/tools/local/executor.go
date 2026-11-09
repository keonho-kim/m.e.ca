// Purpose: Own project and host filesystem handles and captured-output cleanup for local tools.
// Pattern: Filesystem Adapter lifecycle.
// Usage: Composition root creates and closes Executor.
// Related: internal/tools/local/prepare.go, internal/tools/local/paths.go.
package local

import (
	"errors"

	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
	"github.com/keonho-kim/meca/internal/tools/local/process"
)

type Executor struct {
	arguments               argumentCache
	skills                  *skills.Reader
	skillImageInput         atomic.Bool
	retained                artifactRetention
	imageReads              chan struct{}
	redactionVersion        func() int
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

type Options struct {
	Skills           *skills.Reader
	Images           chan struct{}
	RedactRange      func(string, int, int) (string, error)
	RedactionVersion func() int
}

func NewExecutor(project, home, private string, redact func(string) string, options Options) (*Executor, error) {
	if redact == nil {
		return nil, faults.Errorf(diagcatalog.HeadlessASecretRedactorIsRequired)
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
	if options.Images != nil {
		executor.imageReads = options.Images
	}
	executor.redactionVersion = options.RedactionVersion
	executor.skills = options.Skills
	if executor.skills == nil {
		executor.skills = skills.New(project, home, skills.Options{Redact: redact, Check: boundary.SkillTarget, RedactRange: options.RedactRange, Version: options.RedactionVersion})
	}
	executor.Files = fileview.New(project, func(scope policy.Scope, path string) (fileview.Target, error) {
		target, err := executor.resolve(scope, path)
		return fileview.Target{Absolute: target.absolute, Relative: target.relative, Root: target.root}, err
	}, func(text string) string { return executor.redact(text) })
	executor.commands.Save = executor.archiveCommand
	executor.artifacts, executor.artifactLease, err = newArtifacts()
	if err != nil {
		return nil, errors.Join(err, executor.Close())
	}
	executor.commands.CaptureDirectory = executor.artifacts
	return executor, nil
}

func (executor *Executor) Close() error {
	executor.arguments.clear()
	executor.liveOutputs.releaseProjections()
	executor.commands.End()
	executor.liveOutputs.clear()
	err := errors.Join(executor.projectRoot.Close(), executor.systemRoot.Close())
	if executor.artifacts != "" {
		err = errors.Join(err, os.RemoveAll(executor.artifacts))
	}
	executor.retained.clear()
	if executor.artifactLease != nil {
		err = errors.Join(err, executor.artifactLease.Close())
		executor.artifactLease = nil
	}
	return err
}
