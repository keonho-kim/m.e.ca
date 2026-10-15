// Purpose: Require observed verification between repeated changes to the same file.
// Pattern: Pure state transitions.
// Usage: Run observes changed files and clears edit counts after a completed check command.
// Related: internal/agent/run/observation.go, internal/agent/run/action.go.
package progress

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"slices"
)

type Revisions struct{ edits map[string]int }

func (r *Revisions) Observe(changes []toolapi.FileChange) {
	if r.edits == nil {
		r.edits = map[string]int{}
	}
	counted := map[string]bool{}
	for _, change := range changes {
		if change.Path == "" || change.BeforeDigest != "" && change.BeforeDigest == change.AfterDigest || counted[change.Path] {
			continue
		}
		r.edits[change.Path]++
		counted[change.Path] = true
	}
}
func (r *Revisions) Due() string {
	var paths []string
	for path, count := range r.edits {
		if count >= 3 {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}
func (r *Revisions) Checked() { clear(r.edits) }
