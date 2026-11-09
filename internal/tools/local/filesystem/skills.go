// Purpose: Allow dedicated reads of user skills without opening other private application state.
// Pattern: Narrow filesystem access exception.
// Usage: The skill reader calls SkillTarget before pinning each directory.
// Related: internal/tools/local/filesystem/boundary.go, internal/tools/local/skills.go.
package filesystem

import (
	"path/filepath"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (b *Boundary) SkillTarget(path string) error {
	if err := b.Check(); err != nil {
		return err
	}
	actual, err := Canonical(path)
	if err != nil {
		return err
	}
	state, err := Canonical(filepath.Join(b.home, ".meca"))
	if err != nil {
		return err
	}
	// Keep the skills tail literal: a symlink to another private directory
	// must not turn that destination into a permitted application-state root.
	allowed := filepath.Join(state, "skills")
	for _, protected := range b.protected {
		if Within(protected, actual) && !(protected == state && Within(allowed, actual)) {
			return faults.Errorf(diagcatalog.FilesystemProtectedSkillPath)
		}
	}
	return nil
}
