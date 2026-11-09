// Purpose: Keep queued image snapshots within the collection budget used by draft restoration.
// Pattern: Event-loop-owned admission rule.
// Usage: reserveMessage appends through State.Append before clearing the current composer.
// Related: internal/tui/composer/queue/state.go, internal/tui/session/pending.go.
package queue

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const MaxTextBytes = 32 << 20

func (s *State) checkText(candidate requestinput.Request, replace int) error {
	size := len(candidate.Text)
	for _, path := range candidate.Paths {
		size += len(path)
	}
	for i, request := range s.Messages {
		if i != replace {
			size += len(request.Text)
			for _, path := range request.Paths {
				size += len(path)
			}
		}
	}
	if size > MaxTextBytes {
		return faults.New(diagcatalog.ResourceExhausted, diagcatalog.ServerQueueBytes)
	}
	return nil
}

func (s *State) Append(request requestinput.Request) error {
	if err := s.checkText(request, -1); err != nil {
		return err
	}
	if len(request.Images) > 0 {
		images := append([]media.Image(nil), request.Images...)
		for _, queued := range s.Messages {
			images = append(images, queued.Images...)
		}
		if err := media.CheckInput(images); err != nil {
			return err
		}
	}
	s.Messages = append(s.Messages, request)
	return nil
}
