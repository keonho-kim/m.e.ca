// Purpose: Own validated filesystem roots and their cleanup for local tools.
// Pattern: Filesystem Adapter lifecycle.
// Usage: Composition root creates and closes Executor.
// Related: internal/tools/local/prepare.go, internal/tools/local/paths.go.
package local

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/tools/local/anchors"
	"os"
	"path/filepath"
)

type Executor struct {
	reads                   anchors.Store
	project, home, private  string
	projectRoot, systemRoot *os.Root
	redact                  func(string) string
}

func NewExecutor(project, home, private string, redact func(string) string) (*Executor, error) {
	if redact == nil {
		return nil, fmt.Errorf("a secret redactor is required")
	}
	for _, path := range []string{project, home, private} {
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("filesystem roots must be absolute")
		}
	}
	project, err := realPath(project)
	if err != nil {
		return nil, err
	}
	home, err = realPath(home)
	if err != nil {
		return nil, err
	}
	private, err = realPath(private)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(project)
	if err != nil {
		return nil, err
	}
	system, err := os.OpenRoot(filepath.VolumeName(project) + string(filepath.Separator))
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	return &Executor{project: project, home: home, private: private, projectRoot: root, systemRoot: system, redact: redact}, nil
}

func (executor *Executor) Close() error {
	executor.reads.Clear()
	return errors.Join(executor.projectRoot.Close(), executor.systemRoot.Close())
}
