// Purpose: Project actual tool results into bounded completion memory.
// Pattern: Completion state adapter.
// Usage: acceptResult observes executed facts after each tool result.
// Related: internal/agent/completion/state.go, internal/agent/run/completion_review.go.
package run

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *execution) observeCompletion(v validatedCall, outcome toolapi.Result, toolErr error) {
	var targets, changed []string
	versions := map[string]string{}
	for _, c := range outcome.Changes {
		targets = append(targets, c.Path)
		versions[c.Path] = c.AfterDigest
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
	if outcome.Process != nil {
		body, _ := json.Marshal(outcome.Process)
		text += "\n" + string(body)
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
	if v.call.Name == "delete_file" {
		kind = "path_missing"
	}
	if v.definition.Effect == policy.Command {
		kind = "command"
	}
	if v.call.Name == "create_directory" && success {
		var dirs struct{ Paths []string }
		if json.Unmarshal(outcome.Content, &dirs) == nil && len(dirs.Paths) > 0 {
			targets = dirs.Paths
		}
	}
	receipt := completion.Receipt{Kind: kind, Versions: versions, Command: excerpt(commandText, 4000), ID: v.call.ID, Tool: v.call.Name, Targets: targets, Success: success, UnknownScope: command, Output: excerpt(s.runner.Redact(text), 2048)}
	s.checkpoint.Work.Observe(receipt, changed, command)
}
func checkCompletion(c Checkpoint) error {
	if len(c.ReportReason) > 256*1024*1024 {
		return fmt.Errorf("invalid report reason")
	}
	return c.Work.Check()
}
