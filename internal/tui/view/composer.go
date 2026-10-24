// Purpose: Reserve composer input, selected paths, candidates and a transient notice without overlap.
// Pattern: Pure layout calculation.
// Usage: Chat rendering and viewport scrolling share these positions.
// Related: internal/tui/view/chat.go, internal/tui/session/render_chat.go.
package view

type ComposerLayout struct{ InputY, ReferenceY, ReferenceRows, CandidateY, CandidateEnd, TranscriptRows int }

func Composer(rows, height int, choices Candidates, notice string, references int) ComposerLayout {
	inputY := height - rows
	noticeRows := 0
	if notice != "" && inputY > 1 {
		noticeRows = 1
	}
	references = min(references, max(0, inputY-noticeRows-1))
	referenceY := inputY - noticeRows - references
	count := 0
	if choices.Open {
		count = min(MaxCandidateRows, max(1, len(choices.Labels)), max(0, referenceY-1))
	}
	end := referenceY
	candidateY := end - count
	return ComposerLayout{InputY: inputY, ReferenceY: referenceY, ReferenceRows: references, CandidateY: candidateY, CandidateEnd: end, TranscriptRows: max(0, candidateY-2)}
}
