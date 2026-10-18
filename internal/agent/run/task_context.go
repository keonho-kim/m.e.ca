// Purpose: Project execution observations into task information without exposing control state.
// Pattern: Pure context projection.
// Usage: Main-agent requests receive relevant results, remaining work and prior interrupted requests.
// Related: internal/agent/run/context.go, internal/agent/completion/state.go.
package run

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	contextbuild "github.com/keonho-kim/v-code/internal/agent/context"
)

type taskResult struct {
	HistoricalCommand  bool     `json:"historical_command,omitempty"`
	BeforeLaterChanges bool     `json:"before_later_changes,omitempty"`
	Files              []string `json:"files,omitempty"`
	Command            string   `json:"command,omitempty"`
	Result             string   `json:"result"`
	Success            bool     `json:"success"`
	Current            *bool    `json:"current,omitempty"`
}
type taskFacts struct {
	CurrentFiles    []contextbuild.FileObservation `json:"current_files,omitempty"`
	ProjectRoot     string                         `json:"project_root,omitempty"`
	Next            string                         `json:"next,omitempty"`
	Results         []taskResult                   `json:"results,omitempty"`
	LastChange      string                         `json:"last_change,omitempty"`
	Problem         string                         `json:"problem,omitempty"`
	PreviousRequest string                         `json:"previous_request,omitempty"`
}

func taskResults(state completion.State) []taskResult {
	var out []taskResult
	for _, r := range state.Receipts {
		output := r.Output
		var current *bool
		if r.Kind != "command" {
			value := !r.Stale
			current = &value
		}
		out = append(out, taskResult{Files: r.Targets, Command: r.Command, Result: output, Success: r.Success, Current: current, HistoricalCommand: r.Kind == "command", BeforeLaterChanges: r.BeforeLaterChanges})
	}
	return out
}
func (s *execution) taskContext() (json.RawMessage, error) {
	facts := taskFacts{CurrentFiles: contextbuild.Files(s.checkpoint.Work.CurrentFiles), ProjectRoot: s.runner.ProjectRoot, LastChange: s.checkpoint.LastChange, Problem: s.checkpoint.Blocker, Next: s.checkpoint.Observation}
	if s.runner.Redact != nil {
		facts.ProjectRoot = s.runner.Redact(facts.ProjectRoot)
	}
	background := s.workBackground()
	if len(background.EarlierRequests) > 0 {
		facts.PreviousRequest = background.EarlierRequests[len(background.EarlierRequests)-1]
	}
	if s.memoryContext != "" {
		facts.Results = s.missingResults(s.checkpoint.Work)
	}
	if previous := s.runner.PreviousRun; previous != nil && (previous.Status == Interrupted || previous.Status == Stopped) {
		facts.PreviousRequest = previous.Objective
		if facts.Problem == "" && facts.Next == "" {
			facts.Problem = previous.Blocker
			if facts.Problem == "" {
				facts.Problem = previous.Observation
			}
		}
	}
	return json.Marshal(facts)
}
