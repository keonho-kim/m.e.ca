// Purpose: Own ordered follow-up requests, one editable reservation and its saved draft.
// Pattern: Event-loop-owned queue state transitions.
// Usage: The session routes input and worker completion into this state without background mutation.
// Related: internal/tui/session/pending.go, internal/tui/session/pending_edit.go, internal/tui/input/editor.go.
package queue

import (
	"strings"

	"github.com/keonho-kim/meca/internal/tui/input"
)

type Completion uint8

const (
	Running Completion = iota
	Succeeded
	Stopped
)

type Navigation uint8

const (
	Unchanged Navigation = iota
	Focused
	Unsaved
	Close
)

type Edit struct {
	Index      int
	Draft      input.Editor
	Completion Completion
	Reveal     bool
}

// State is owned by the terminal event loop; Messages must remain ordered while Edit is active.
type State struct {
	Messages []string
	Edit     *Edit
}

func (s *State) Changed(text string) bool {
	return s.Edit != nil && text != s.Messages[s.Edit.Index]
}

func (s *State) Navigate(delta int, editor *input.Editor) Navigation {
	if s.Edit == nil {
		if delta >= 0 || len(s.Messages) == 0 {
			return Unchanged
		}
		s.Edit = &Edit{Index: len(s.Messages) - 1, Draft: *editor, Reveal: true}
		editor.Set(s.Messages[s.Edit.Index])
		return Focused
	}
	next := s.Edit.Index + delta
	if next < 0 {
		return Unchanged
	}
	if s.Changed(editor.String()) {
		return Unsaved
	}
	if next >= len(s.Messages) {
		return Close
	}
	s.Edit.Index, s.Edit.Reveal = next, true
	editor.Set(s.Messages[next])
	return Focused
}

func (s *State) Save(text string) bool {
	if s.Edit == nil || strings.TrimSpace(text) == "" {
		return false
	}
	s.Messages[s.Edit.Index] = text
	return true
}

func (s *State) DeleteSelected() {
	if s.Edit == nil {
		return
	}
	index := s.Edit.Index
	copy(s.Messages[index:], s.Messages[index+1:])
	s.Messages[len(s.Messages)-1] = ""
	s.Messages = s.Messages[:len(s.Messages)-1]
	if len(s.Messages) == 0 {
		s.Messages = nil
	}
}

func (s *State) RestoreEditor(editor *input.Editor) Completion {
	edit := s.Edit
	if edit == nil {
		return Running
	}
	s.Edit = nil
	*editor = edit.Draft
	return edit.Completion
}

func (s *State) Drain(editor *input.Editor) {
	if len(s.Messages) == 0 {
		return
	}
	text := strings.Join(s.Messages, "\n\n")
	if draft := editor.String(); draft != "" {
		text += "\n\n" + draft
	}
	editor.Set(text)
	clear(s.Messages)
	s.Messages = nil
}

func (s *State) Shift() {
	s.Messages[0] = ""
	s.Messages = s.Messages[1:]
	if len(s.Messages) == 0 {
		s.Messages = nil
	}
}
