// Purpose: Carry compact observed files and command outcomes for run reports.
// Pattern: Runtime fact values.
// Usage: Run results carry Facts independently of conversation history.
// Related: internal/agent/run/observations.go, internal/agent/run/report.go.
package tool

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

type FileFact struct {
	Observation string `json:"observation,omitempty"`
	SourceID    string `json:"source_id,omitempty"`
	ObservedRun string `json:"observed_run,omitempty"`
	Revision    int    `json:"revision,omitempty"`
	Path        string `json:"path"`
	Version     string `json:"version"`
	Changed     bool   `json:"changed"`
	Stale       bool   `json:"stale"`
}

type CommandFact struct {
	CallID             string `json:"call_id,omitempty"`
	ObservedRun        string `json:"observed_run,omitempty"`
	Revision           int    `json:"revision"`
	BeforeLaterChanges bool   `json:"before_later_changes"`
	Command            string `json:"command"`
	Status             string `json:"status"`
	ExitCode           *int   `json:"exit_code,omitempty"`
	Excerpt            bool   `json:"excerpt,omitempty"`
}

type Facts struct {
	Files    []FileFact    `json:"files,omitempty"`
	Commands []CommandFact `json:"-"`
}

func (f Facts) Check() error {
	body, err := json.Marshal(f)
	if err != nil || len(body) > 16*1024*1024 {
		return fmt.Errorf("execution report facts exceed 16 MiB; source records remain stored")
	}
	for _, v := range f.Files {
		if v.Revision < 0 || len(v.SourceID) > 256 || len(v.ObservedRun) > 256 || (v.Observation != "" && v.Observation != "present" && v.Observation != "missing" && v.Observation != "unavailable") {
			return fmt.Errorf("invalid file observation provenance")
		}
		if v.Path == "" || len(v.Path) > 4096 || len(v.Version) > 128 || !utf8.ValidString(v.Path) || !utf8.ValidString(v.Version) {
			return fmt.Errorf("invalid reported file observation")
		}
	}
	for _, v := range f.Commands {
		if v.Revision < 0 || len(v.ObservedRun) > 256 || !utf8.ValidString(v.ObservedRun) || len(v.CallID) > 128 || !utf8.ValidString(v.CallID) {
			return fmt.Errorf("invalid reported command ordering")
		}
		if v.Command == "" || !utf8.ValidString(v.Command) {
			return fmt.Errorf("invalid reported command observation")
		}
	}
	return nil
}
