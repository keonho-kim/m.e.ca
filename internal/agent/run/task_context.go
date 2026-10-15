// Purpose: Project execution observations into task information without exposing control state.
// Pattern: Pure context projection.
// Usage: Main-agent requests receive relevant results, remaining work and prior interrupted requests.
// Related: internal/agent/run/context.go, internal/agent/completion/state.go.
package run

import (
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/completion"
)

type taskResult struct {
	Files   []string `json:"files,omitempty"`
	Command string   `json:"command,omitempty"`
	Result  string   `json:"result"`
	Success bool     `json:"success"`
	Current *bool    `json:"current,omitempty"`
}
type taskFacts struct {
	Next            string       `json:"next,omitempty"`
	Results         []taskResult `json:"results,omitempty"`
	LastChange      string       `json:"last_change,omitempty"`
	Problem         string       `json:"problem,omitempty"`
	Remaining       []string     `json:"remaining,omitempty"`
	PreviousRequest string       `json:"previous_request,omitempty"`
}

func taskResults(state completion.State) []taskResult {
	var out []taskResult
	for _, r := range state.Receipts {
		var current *bool
		if r.Kind != "command" {
			value := !r.Stale
			current = &value
		}
		out = append(out, taskResult{Files: r.Targets, Command: r.Command, Result: r.Output, Success: r.Success, Current: current})
	}
	return out
}
func (s *execution) taskContext() (json.RawMessage, error) {
	facts := taskFacts{LastChange: s.checkpoint.LastChange, Problem: s.checkpoint.Blocker, Next: s.checkpoint.Observation}
	if s.memoryContext != "" {
		facts.Results = taskResults(s.checkpoint.Work)
	}
	if verdict := s.checkpoint.Work.Verdict; verdict != nil && verdict.Decision == "continue" {
		facts.Remaining = verdict.Unmet
	}
	if previous := s.runner.PreviousRun; previous != nil && (previous.Status == Interrupted || previous.Status == Stopped || previous.Status == Unverified) {
		facts.PreviousRequest = previous.Objective
		facts.Results = append(taskResults(previous.Work), facts.Results...)
		if facts.Problem == "" {
			facts.Problem = previous.Blocker
			if facts.Problem == "" {
				facts.Problem = previous.Observation
			}
		}
	}
	return json.Marshal(facts)
}
