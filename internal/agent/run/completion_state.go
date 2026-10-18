// Purpose: Project actual tool results into bounded completion memory.
// Pattern: Completion state adapter.
// Usage: acceptResult observes executed facts after each tool result.
// Related: internal/agent/completion/state.go, internal/agent/run/finish_response.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"maps"
	"strings"
)

func (s *execution) observeCompletion(ctx context.Context, v validatedCall, outcome toolapi.Result, toolErr error) error {
	var targets, changed []string
	versions := maps.Clone(outcome.Versions)
	if versions == nil {
		versions = map[string]string{}
	}
	for _, c := range outcome.Changes {
		targets = append(targets, c.Path)
		if c.AfterDigest != "" {
			versions[c.Path] = c.AfterDigest
		}
		if c.BeforeDigest != c.AfterDigest || c.Destination != "" {
			changed = append(changed, c.Path)
		}
		if c.Destination != "" {
			targets = append(targets, c.Destination)
			changed = append(changed, c.Destination)
		}
	}
	if len(targets) == 0 && v.definition.Effect != policy.Command && outcome.Target != "" {
		targets = []string{outcome.Target}
	}
	if len(targets) == 0 && v.definition.Effect != policy.Command {
		var path string
		if json.Unmarshal(v.arguments["path"], &path) == nil && path != "" {
			targets = []string{path}
		}
	}
	var commandText string
	if v.definition.Effect == policy.Command {
		_ = json.Unmarshal(v.arguments["command"], &commandText)
	}
	text := outcome.Summary
	if toolErr != nil {
		text += "\n" + toolErr.Error()
	}
	// Preserve actual exit/output meaning (including zero matches), not only success.
	if len(outcome.Content) > 0 {
		text += "\n" + string(outcome.Content)
	}
	success := outcome.Started && !outcome.Failed && toolErr == nil
	if p := outcome.Process; p != nil && (p.Status != "exited" || p.ExitCode != 0) {
		success = false
	}
	command := v.definition.Effect == policy.Command && outcome.Started
	kind := "content"
	if v.call.Name == "create_directory" {
		kind = "directory_exists"
	}
	if v.definition.Effect == policy.Command {
		kind = "command"
	}
	if v.call.Name == "create_directory" && success {
		var dirs struct{ Paths []string }
		if json.Unmarshal(outcome.Content, &dirs) == nil && len(dirs.Paths) > 0 {
			targets = dirs.Paths
			for _, path := range dirs.Paths {
				versions[path] = "directory"
			}
		}
	}
	failure := outcome.Failure
	if failure == nil {
		failure = toolapi.FailureOf(toolErr)
	}
	receipt := completion.Receipt{Failure: failure, Changed: len(changed) > 0, Kind: kind, Versions: versions, Command: commandText, ID: v.call.ID, Tool: v.call.Name, Targets: targets, Success: success, Output: s.runner.Redact(strings.TrimLeft(text, "\r\n"))}
	receipt.ObservedRun, receipt.Revision = s.checkpoint.ID, s.checkpoint.ChangeRevision
	if commandText != "" {
		receipt.ProcessStatus = "not_started"
		if outcome.Started {
			receipt.ProcessStatus = "unknown"
		}
		if p := outcome.Process; p != nil {
			receipt.ProcessStatus = p.Status
			receipt.ExitCode = &p.ExitCode
		}
	}
	s.checkpoint.Work.Observe(receipt, changed, command)
	if command && changedFiles(outcome) {
		s.checkpoint.LastChange = excerpt(s.runner.Redact(commandChangeNote(outcome.Changes)), 768)
	}
	if command {
		if outcome.Versions != nil {
			s.checkpoint.Work.ReconcileFiles(outcome.Versions)
		} else if err := s.refreshFileFacts(ctx); err != nil {
			return err
		}
	}
	return s.checkpoint.Work.Check()
}
func checkCompletion(c Checkpoint) error {
	if len(c.ReportReason) > 256*1024*1024 {
		return fmt.Errorf("invalid report reason")
	}
	return c.Work.Check()
}
