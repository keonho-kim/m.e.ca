// Purpose: Reserve a bottom-anchored composer with candidates and a transient notice above it.
// Pattern: Pure layout calculation.
// Usage: Chat rendering and viewport scrolling share these positions.
// Related: internal/tui/view/chat.go, internal/tui/session/render_chat.go.
package view

import "github.com/keonho-kim/v-code/internal/tui/input"

type ComposerLayout struct{ InputY, InputRows, CandidateY, CandidateEnd, TranscriptRows int }

func Composer(editor input.Editor, width, height, maxRows int, choices Candidates, notice string) ComposerLayout {
	rows := InputRows(editor, max(1, width), min(maxRows, max(1, height-1)))
	inputY := height - rows
	noticeRows := 0
	if notice != "" && inputY > 1 {
		noticeRows = 1
	}
	count := 0
	if choices.Open {
		count = min(MaxCandidateRows, max(1, len(choices.Labels)), max(0, inputY-noticeRows-1))
	}
	end := inputY - noticeRows
	candidateY := end - count
	return ComposerLayout{InputY: inputY, InputRows: rows, CandidateY: candidateY, CandidateEnd: end, TranscriptRows: max(0, candidateY-2)}
}
