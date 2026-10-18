// Purpose: Own validated filesystem roots and their cleanup for local tools.
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
	projectRoot, systemRoot *os.Root
	redact                  func(string) string
	boundary                *filesystem.Boundary
	writeRoots              map[string]*os.Root
}

func NewExecutor(project, home, private string, writable []string, redact func(string) string) (*Executor, error) {
	if redact == nil {
		return nil, fmt.Errorf("a secret redactor is required")
	}
	boundary, err := filesystem.New(project, home, private, writable)
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
	executor := &Executor{project: project, home: home, projectRoot: root, systemRoot: system, redact: redact, boundary: boundary, writeRoots: map[string]*os.Root{project: root}}
	for _, path := range boundary.Writes() {
		if path == project {
			continue
		}
		handle, err := os.OpenRoot(path)
		if err != nil {
			return nil, errors.Join(err, executor.Close())
		}
		executor.writeRoots[path] = handle
	}
	return executor, nil
}

func (executor *Executor) Close() error {
	var err error
	for _, root := range executor.writeRoots {
		err = errors.Join(err, root.Close())
	}
	return errors.Join(err, executor.systemRoot.Close())
}
