// Purpose: Pin the project directory and guard private paths for dedicated file tools.
// Pattern: Filesystem access policy.
// Usage: Local Executor owns this guard; host shell execution uses the OS user permissions.
// Related: internal/tools/local/filesystem/paths.go, internal/tools/local/executor.go.
package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/keonho-kim/meca/internal/config/statefs"
)

type Boundary struct {
	project, home string
	projectInfo   os.FileInfo
	protected     []string
}

func New(project, home, private string) (*Boundary, error) {
	var err error
	project, err = Canonical(project)
	if err != nil {
		return nil, err
	}
	home, err = Canonical(home)
	if err != nil {
		return nil, err
	}
	private, err = Canonical(private)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(project)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project root must be a directory")
	}
	b := &Boundary{project: project, home: home, projectInfo: info, protected: []string{private}}
	for _, name := range []string{statefs.DirectoryName, ".v", ".ssh", ".gnupg", ".aws", ".azure", ".kube", ".docker", ".colima", ".orbstack", ".config/gcloud", ".config/gh", ".config/op", ".netrc", "Library/Keychains"} {
		path, err := Canonical(filepath.Join(home, name))
		if err != nil {
			return nil, err
		}
		if !slices.Contains(b.protected, path) {
			b.protected = append(b.protected, path)
		}
	}
	return b, nil
}

func (b *Boundary) Project() string { return b.project }
func (b *Boundary) Home() string    { return b.home }

func (b *Boundary) Check() error {
	path, err := Canonical(b.project)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if path != b.project || !os.SameFile(info, b.projectInfo) {
		return fmt.Errorf("project root changed: %s", b.project)
	}
	return nil
}

func (b *Boundary) Movable(path string) error {
	if Within(path, b.project) {
		return fmt.Errorf("cannot move the project root or its ancestor: %s", path)
	}
	for _, secret := range b.protected {
		if Within(path, secret) {
			return fmt.Errorf("cannot move protected path ancestor: %s", path)
		}
	}
	return nil
}
