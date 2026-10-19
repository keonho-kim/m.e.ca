// Purpose: Own project and host filesystem handles and captured-output cleanup for local tools.
// Pattern: Filesystem Adapter lifecycle.
// Usage: Composition root creates and closes Executor.
// Related: internal/tools/local/prepare.go, internal/tools/local/paths.go.
package local

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/tools/local/filesystem"
	"os"
	"path/filepath"
)

type Executor struct {
	project, home           string
	artifacts               string
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
	executor := &Executor{project: project, home: home, projectRoot: root, systemRoot: system, redact: redact, boundary: boundary}
	executor.artifacts, err = os.MkdirTemp("", "v-output-*")
	if err != nil {
		return nil, errors.Join(err, executor.Close())
	}
	return executor, nil
}

func (executor *Executor) Close() error {
	err := errors.Join(executor.projectRoot.Close(), executor.systemRoot.Close())
	if executor.artifacts != "" {
		err = errors.Join(err, os.RemoveAll(executor.artifacts))
	}
	return err
}
