// Purpose: Project committed execution facts into compact public observations without I/O.
// Pattern: Pure operation-to-message transformation.
// Usage: Peer.observe applies deduplication and commits the returned observation.
// Related: internal/agent/team/observations.go, internal/agent/team/channel/message.go.
package team

import (
	"fmt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"path/filepath"
	"strings"
)

func operationObservation(member, projectRoot string, event runrecord.Event) channel.Message {
	op := event.Operation
	var text strings.Builder
	text.WriteString(op.Name + " " + op.Status)
	var paths []string
	for _, change := range op.Changes {
		for _, path := range []string{change.Path, change.Destination} {
			if path == "" {
				continue
			}
			if !filepath.IsAbs(path) {
				path = filepath.Join(projectRoot, path)
			}
			paths = append(paths, filepath.Clean(path))
		}
		text.WriteString("\n" + change.Action + " " + change.Path)
		if change.Destination != "" {
			text.WriteString(" -> " + change.Destination)
		}
	}
	for _, path := range op.AffectedPaths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(projectRoot, path)
		}
		paths = append(paths, filepath.Clean(path))
		text.WriteString("\nMay have affected: " + path)
	}
	if process := op.Process; process != nil {
		text.WriteString("\nCommand " + process.CommandID + ": " + process.Status)
		if !toolapi.CommandActive(process.Status) {
			fmt.Fprintf(&text, "; exit=%d", process.ExitCode)
		}
		if len(op.Command) <= 4096 {
			text.WriteString("\n" + op.Command)
		}
	}
	if op.Failure != nil {
		text.WriteString("\nFailure: " + op.Failure.Code + "; ask " + member + " for details if relevant.")
		if detail, _, _ := strings.Cut(op.Failure.Detail, "\n"); len(detail) <= 4096 {
			text.WriteString("\n" + detail)
		}
	}
	if event.Reference != nil {
		fmt.Fprintf(&text, "\nOriginal record: %s/%d", event.Reference.Session, event.Reference.Sequence)
	}
	passive := len(paths) == 0 && op.Failure == nil && op.Status == "tool_succeeded"
	if process := op.Process; process != nil && !toolapi.CommandActive(process.Status) && (process.Status != "exited" || process.ExitCode != 0) {
		passive = false
	}
	return channel.Message{From: member, Text: text.String(), Observation: true, Passive: passive, ChangedPaths: paths, Reference: event.Reference}
}
