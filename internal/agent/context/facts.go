// Purpose: Expose useful file and command observations without execution provenance.
// Pattern: Pure model-context projection.
// Usage: Request preparation projects current file observations without provenance.
// Related: internal/agent/tool/facts.go, internal/agent/run/task_context.go.
package context

import "github.com/keonho-kim/v-code/internal/agent/tool"

type FileObservation struct {
	Path     string `json:"path"`
	State    string `json:"state"`
	Changed  bool   `json:"changed,omitempty"`
	Outdated bool   `json:"outdated,omitempty"`
}

func Files(source []tool.FileFact) []FileObservation {
	var files []FileObservation
	for _, f := range source {
		state := f.Observation
		if state == "" {
			state = "present"
			if f.Version == "missing" {
				state = "missing"
			} else if f.Version == "" {
				state = "unavailable"
			}
		}
		files = append(files, FileObservation{f.Path, state, f.Changed, f.Stale})
	}
	return files
}
