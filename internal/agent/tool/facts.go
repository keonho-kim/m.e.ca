// Purpose: Carry compact observed files and command outcomes for run reports.
// Pattern: Runtime fact values.
// Usage: Run results carry Facts independently of conversation history.
// Related: internal/agent/run/observations.go, internal/agent/run/report.go.
package tool

import (
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
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
	CommandID          string "json:\"command_id,omitempty\""
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
	if factsBytes(f) > 16*1024*1024 {
		return faults.Errorf(diagcatalog.ToolExecutionReportFactsExceedMiBSourceRecords)
	}
	for _, v := range f.Files {
		if v.Revision < 0 || len(v.SourceID) > 256 || len(v.ObservedRun) > 256 || (v.Observation != "" && v.Observation != "present" && v.Observation != "missing" && v.Observation != diagcatalog.Unavailable) {
			return faults.Errorf(diagcatalog.ToolInvalidFileObservationProvenance)
		}
		if v.Path == "" || len(v.Path) > 4096 || len(v.Version) > 128 || !utf8.ValidString(v.Path) || !utf8.ValidString(v.Version) {
			return faults.Errorf(diagcatalog.ToolInvalidReportedFileObservation)
		}
	}
	for _, v := range f.Commands {
		if v.Revision < 0 || len(v.ObservedRun) > 256 || !utf8.ValidString(v.ObservedRun) || len(v.CallID) > 128 || !utf8.ValidString(v.CallID) {
			return faults.Errorf(diagcatalog.ToolInvalidReportedCommandOrdering)
		}
		if v.Command == "" || !utf8.ValidString(v.Command) {
			return faults.Errorf(diagcatalog.ToolInvalidReportedCommandObservation)
		}
	}
	return nil
}
