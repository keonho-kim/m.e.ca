// Purpose: Share compact durable execution facts without copying private output or reasoning.
// Pattern: Ordered publication derived from committed operation records.
// Usage: Peer.Emit commits an original reference and its compact fact before presentation.
// Related: internal/agent/team/publication.go, internal/agent/team/contribution/state.go.
package team

import (
	"fmt"
	"path/filepath"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func (p *Peer) observe(event runrecord.Event) (bool, error) {
	op := event.Operation
	if op == nil || len(op.Changes) == 0 && op.Process == nil && (!op.Started || op.Failure == nil) {
		return false, nil
	}
	r := p.room
	r.logMu.Lock()
	defer r.logMu.Unlock()
	if op.Process != nil {
		_, running := p.running[op.Process.CommandID]
		initial := op.Name == "execute_command" && event.Kind != "command_finished"
		if running && op.Process.Status == "running" || !running && !initial {
			return false, nil
		}
	}
	var text strings.Builder
	text.WriteString(op.Name + " " + op.Status)
	var paths []string
	for _, change := range op.Changes {
		for _, path := range []string{change.Path, change.Destination} {
			if path == "" {
				continue
			}
			if !filepath.IsAbs(path) {
				path = filepath.Join(r.options.ProjectRoot, path)
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
			path = filepath.Join(r.options.ProjectRoot, path)
		}
		paths = append(paths, filepath.Clean(path))
		text.WriteString("\nMay have affected: " + path)
	}
	if process := op.Process; process != nil {
		text.WriteString("\nCommand " + process.CommandID + ": " + process.Status)
		if process.Status != "running" {
			fmt.Fprintf(&text, "; exit=%d", process.ExitCode)
		}
		if len(op.Command) <= 4096 {
			text.WriteString("\n" + op.Command)
		}
	}
	if op.Failure != nil {
		text.WriteString("\nFailure: " + op.Failure.Code + "; ask " + p.ID + " for details if relevant.")
		if detail, _, _ := strings.Cut(op.Failure.Detail, "\n"); len(detail) <= 4096 {
			text.WriteString("\n" + detail)
		}
	}
	if event.Reference != nil {
		fmt.Fprintf(&text, "\nOriginal record: %s/%d", event.Reference.Session, event.Reference.Sequence)
	}
	passive := len(paths) == 0 && op.Failure == nil && op.Status == "tool_succeeded"
	if process := op.Process; process != nil && process.Status != "running" && (process.Status != "exited" || process.ExitCode != 0) {
		passive = false
	}
	message := channel.Message{From: p.ID, Text: text.String(), Observation: true, Passive: passive, ChangedPaths: paths, Reference: event.Reference}
	if _, err := r.appendPublic(message); err != nil {
		return false, err
	}
	if process := op.Process; process != nil {
		if process.Status == "running" {
			if p.running == nil {
				p.running = make(map[string]struct{})
			}
			p.running[process.CommandID] = struct{}{}
		} else {
			delete(p.running, process.CommandID)
			if len(p.running) == 0 {
				p.running = nil
			}
		}
	}
	return true, nil
}
