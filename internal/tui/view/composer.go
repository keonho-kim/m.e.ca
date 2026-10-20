// Purpose: Reserve a bottom-anchored composer with candidates and a transient notice above it.
// Pattern: Pure layout calculation.
// Usage: Chat rendering and viewport scrolling share these positions.
// Related: internal/tui/view/chat.go, internal/tui/session/render_chat.go.
package view

type ComposerLayout struct{ InputY, CandidateY, CandidateEnd, TranscriptRows int }

func Composer(rows, height int, choices Candidates, notice string) ComposerLayout {
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
	return ComposerLayout{InputY: inputY, CandidateY: candidateY, CandidateEnd: end, TranscriptRows: max(0, candidateY-2)}
}
