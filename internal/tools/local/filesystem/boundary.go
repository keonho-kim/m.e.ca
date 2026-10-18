// Purpose: Freeze trusted write roots and protected paths independently of Permission.
// Pattern: Filesystem policy module.
// Usage: Composition creates a Boundary; file and process adapters enforce its snapshot.
// Related: internal/tools/local/filesystem/paths.go, internal/tools/local/filesystem/trees_unix.go.
package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

type directory struct {
	path string
	info os.FileInfo
}

type Boundary struct {
	project, home, private string
	writes                 []directory
	protected              []string
}

func New(project, home, private string, extra []string) (*Boundary, error) {
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
	b := &Boundary{project: project, home: home, private: private, protected: []string{private}}
	for _, name := range []string{".v", ".ssh", ".gnupg", ".aws", ".azure", ".kube", ".docker", ".colima", ".orbstack", ".config/gcloud", ".config/gh", ".config/op", ".netrc", "Library/Keychains"} {
		path, err := Canonical(filepath.Join(home, name))
		if err != nil {
			return nil, err
		}
		if !slices.Contains(b.protected, path) {
			b.protected = append(b.protected, path)
		}
	}
	for i, input := range append([]string{project}, extra...) {
		if !filepath.IsAbs(input) {
			return nil, fmt.Errorf("write root must be absolute: %s", input)
		}
		path, err := Canonical(input)
		if err != nil {
			return nil, err
		}
		if unsafeRoot(path, home) {
			return nil, fmt.Errorf("unsafe write root: %s", path)
		}
		for _, secret := range b.protected {
			if Within(secret, path) || (i > 0 && Within(path, secret)) {
				return nil, fmt.Errorf("write root overlaps protected path: %s", path)
			}
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("write root %s: %w", path, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("write root must be a directory: %s", path)
		}
		duplicate := slices.ContainsFunc(b.writes, func(d directory) bool { return Within(d.path, path) })
		for _, d := range b.writes {
			if path != d.path && Within(path, d.path) {
				return nil, fmt.Errorf("write root overlaps another root: %s", path)
			}
		}
		if !duplicate {
			b.writes = append(b.writes, directory{path, info})
		}
	}
	return b, nil
}
func (b *Boundary) Project() string     { return b.project }
func (b *Boundary) Home() string        { return b.home }
func (b *Boundary) Private() string     { return b.private }
func (b *Boundary) Protected() []string { return slices.Clone(b.protected) }
func (b *Boundary) Writes() []string {
	paths := make([]string, len(b.writes))
	for i, d := range b.writes {
		paths[i] = d.path
	}
	return paths
}
func (b *Boundary) Check() error {
	for _, d := range b.writes {
		path, err := filepath.EvalSymlinks(d.path)
		if err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if path != d.path || !os.SameFile(info, d.info) {
			return fmt.Errorf("write root changed: %s", d.path)
		}
	}
	return nil
}
func unsafeRoot(path, home string) bool {
	if path == "/" || Within(path, home) || path == "/tmp" || path == "/var" || path == "/private/var" || path == "/private/tmp" {
		return true
	}
	for _, system := range []string{"/bin", "/sbin", "/usr", "/lib", "/lib64", "/etc", "/System", "/Library", "/opt", "/Applications", "/dev", "/proc", "/sys", "/run", "/private/etc", "/private/var/db", "/var/lib"} {
		if Within(system, path) {
			return true
		}
	}
	return false
}
