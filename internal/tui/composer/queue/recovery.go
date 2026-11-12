// Purpose: Restore accepted requests only when their merged text fits the editable-input budget.
// Pattern: Explicit recovery transition with a paused overflow queue.
// Usage: Interrupted steering and queued requests share RestoreText; Enter explicitly resumes overflow.
// Related: internal/tui/session/steering.go, internal/tui/session/pending.go, internal/tui/input/editor.go.
package queue

import (
	"strings"

	"github.com/keonho-kim/meca/internal/agent/media"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"github.com/keonho-kim/meca/internal/tui/input"
)

// RestoreText does not allocate or mutate the editor when the full draft cannot fit.
func RestoreText(editor *input.Editor, requests []requestinput.Request) bool {
	if len(requests) == 0 {
		return true
	}
	draft := editor.String()
	size := len(draft)
	if draft != "" {
		size += 2
	}
	for i, request := range requests {
		if i > 0 {
			size += 2
		}
		size += request.TextBytes()
		if size > input.MaxBytes {
			return false
		}
	}
	var body strings.Builder
	body.Grow(size)
	for i, request := range requests {
		if i > 0 {
			body.WriteString("\n\n")
		}
		body.WriteString(request.Body())
	}
	if draft != "" {
		body.WriteString("\n\n")
		body.WriteString(draft)
	}
	editor.Set(body.String())
	return true
}

// Recover retains already accepted snapshots; it never admits new input or grows their bodies.
func (s *State) Recover(requests []requestinput.Request) {
	s.Messages = append(s.Messages, requests...)
	s.Paused = true
}

func (s *State) Drain(editor *input.Editor, draftImages ...media.Image) []media.Image {
	if len(s.Messages) == 0 {
		s.Paused = false
		return draftImages
	}
	if !RestoreText(editor, s.Messages) {
		s.Paused = true
		return draftImages
	}
	var images []media.Image
	for _, request := range s.Messages {
		images = append(images, request.Images...)
	}
	clear(s.Messages)
	s.Messages = nil
	s.Paused = false
	return append(images, draftImages...)
}
