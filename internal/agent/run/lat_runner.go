// Purpose: Compose the existing main runner as one isolated read-only LAT phase.
// Pattern: Runtime composition with shared budget and separately owned tools and journal.
// Usage: runLATPhase retains lifetime ownership while latRunner assembles only dependency values.
// Related: internal/agent/run/lat_fork.go, internal/agent/lat/transcript/journal.go.
package run

import (
	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/lat"
	"github.com/keonho-kim/meca/internal/agent/lat/transcript"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) latRunner(instructions string, role inference.Role, packet lat.Packet, definitions []toolapi.Definition, tools toolapi.Executor, journal *transcript.Journal) Runner {
	child := s.runner
	child.LAT, child.PreviousRun, child.ReviewMemory = false, nil, nil
	child.Generator, child.Tools, child.Journal = s.review.generator, tools, journal
	child.fork = &forkConfig{role: role, maxOutput: packet.MaxOutput, budget: s.usage, system: s.system + "\n\n" + instructions, definitions: definitions}
	settings := compact.Defaults()
	if s.runner.Memory != nil {
		settings = s.runner.Memory.Settings
	}
	child.Memory = &compact.Controller{ContextWindow: s.latWindow(), Inference: child.Inference, Repository: journal, Session: child.Session, Model: child.Model, Prompt: s.latTemplates().Compaction, Settings: settings, Redact: child.Redact}
	return child
}
