// Purpose: Classify mandatory approvals and blocked targets for explicit mutations.
// Pattern: Pure policy functions.
// Usage: Shell analysis and dedicated file operations assess canonical targets here.
// Related: internal/policy/policy.go, internal/policy/command/analyze.go.
package policy

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type Safety struct {
	Reasons []string
	Targets []string
}

func (s *Safety) Ask(reason string) {
	if !slices.Contains(s.Reasons, reason) {
		s.Reasons = append(s.Reasons, reason)
	}
}

func (s *Safety) Merge(other Safety) {
	for _, reason := range other.Reasons {
		s.Ask(reason)
	}
	for _, path := range other.Targets {
		if !slices.Contains(s.Targets, path) {
			s.Targets = append(s.Targets, path)
		}
	}
}

func contains(parent, path string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && (relative == "." || filepath.IsLocal(relative))
}

// Mutation receives already canonical paths. Resolution and symlink semantics
// belong to the filesystem adapter, not this deterministic rule.
func Mutation(project, home, path string, removal bool) (Safety, error) {
	return mutation(project, home, path, removal, false)
}

// MutationWithin checks a known directory bound for runtime-selected descendants.
// It does not claim that any individual child has been resolved or observed.
func MutationWithin(project, home, directory string, removal bool) (Safety, error) {
	return mutation(project, home, directory, removal, true)
}

func mutation(project, home, path string, removal, contents bool) (Safety, error) {
	var result Safety
	if !filepath.IsAbs(path) {
		return result, fmt.Errorf("mutation target is not absolute")
	}
	path = filepath.Clean(path)
	if path == "/dev/null" && !removal {
		return result, nil
	}
	protected := []string{"/bin", "/sbin", "/etc", "/usr", "/lib", "/lib64", "/boot", "/dev", "/proc", "/sys", "/System", "/Library", "/private/etc", "/var/lib", "/var/db", "/var/run", "/private/var/db", "/private/var/run"}
	if home != "" {
		protected = append(protected, filepath.Join(home, "bin"))
	}
	for _, root := range protected {
		if contains(root, path) {
			return result, fmt.Errorf("operation blocked: protected system target %s", path)
		}
	}
	if path == "/" || path == home || !contents && path == project || removal && path != project && contains(path, project) {
		return result, fmt.Errorf("operation blocked: protected root %s", path)
	}
	for _, root := range []string{"/Users", "/home", "/tmp", "/var", "/private", "/private/var", "/private/tmp", "/opt"} {
		if path == root {
			return result, fmt.Errorf("operation blocked: protected root %s", path)
		}
	}
	result.Targets = []string{path}
	if removal {
		result.Ask("Deletion or move requires approval")
	}
	if !contains(project, path) {
		result.Ask("Mutation outside the project requires approval")
	}
	return result, nil
}

func (s Safety) Description() string {
	if len(s.Reasons) == 0 {
		return ""
	}
	text := strings.Join(s.Reasons, "\n")
	if len(s.Targets) > 0 {
		text += "\nTargets:\n" + strings.Join(s.Targets, "\n")
	}
	return text
}
