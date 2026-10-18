// Purpose: Observe known filesystem effects around one shell command on the run owner.
// Pattern: Execution observation use case.
// Usage: startExecution captures prior versions; recordExecution records actual post-command changes.
// Related: internal/agent/run/file_facts.go, internal/agent/run/execution_result.go.
package run

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func (s *execution) commandVersions(ctx context.Context, paths []string) (map[string]string, error) {
	reader, ok := s.runner.Tools.(fileVersions)
	if !ok || len(paths) == 0 {
		return nil, nil
	}
	return reader.FileVersions(ctx, s.runner.Settings, paths)
}
func (s *execution) observeCommandFiles(ctx context.Context, ready *readyExecution, result *toolapi.Result) error {
	paths := slices.Sorted(maps.Keys(ready.beforeVersions))
	if len(paths) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	after, err := s.commandVersions(ctx, paths)
	if result.Versions == nil {
		result.Versions = map[string]string{}
	}
	maps.Copy(result.Versions, after)
	for _, path := range paths {
		old, next := ready.beforeVersions[path], after[path]
		if old == "" || next == "" || old == next {
			continue
		}
		action := "command"
		if next == "missing" {
			action = "delete"
		}
		result.Changes = append(result.Changes, toolapi.FileChange{Action: action, Path: path, BeforeDigest: old, AfterDigest: next})
	}
	if err != nil {
		return fmt.Errorf("command finished; known-file observations incomplete: %w", err)
	}
	return nil
}

func commandChangeNote(changes []toolapi.FileChange) string {
	var notes []string
	for _, change := range changes {
		if len(notes) == 4 {
			break
		}
		action := "Updated"
		if change.AfterDigest == "missing" {
			action = "Deleted"
		}
		notes = append(notes, action+" "+change.Path)
	}
	return "execute_command: " + strings.Join(notes, "; ")
}
