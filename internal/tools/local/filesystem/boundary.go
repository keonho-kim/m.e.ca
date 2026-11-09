// Purpose: Pin the project directory and guard private paths for dedicated file tools.
// Pattern: Filesystem access policy.
// Usage: Local Executor owns this guard; host shell execution uses the OS user permissions.
// Related: internal/tools/local/filesystem/paths.go, internal/tools/local/executor.go.
package filesystem

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
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
		return nil, faults.Errorf(diagcatalog.FilesystemProjectRootMustBeADirectory)
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
		return faults.Errorf(diagcatalog.FilesystemProjectRootChanged, b.project)
	}
	return nil
}
