// Purpose: Preserve public collaboration references and self-reported work without copying message bodies.
// Pattern: Immutable event metadata.
// Usage: Team publication creates metadata; SQLite and terminal projections restore the same values.
// Related: internal/agent/record/events.go, internal/agent/team/channel/message.go.
package record

import (
	"slices"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type TeamNote struct {
	ReplyTo     int64     `json:"reply_to,omitempty"`
	ReplyAuthor string    `json:"reply_author,omitempty"`
	Mentions    []string  `json:"mentions,omitempty"`
	BasedOn     int64     `json:"based_on,omitempty"`
	Work        *TeamWork `json:"work,omitempty"`
}

type TeamWork struct {
	Paths    []string `json:"paths"`
	Contract string   `json:"contract,omitempty"`
	Revision uint64   `json:"revision"`
}

// Clone transfers ownership at the public append boundary; readers share the frozen result.
func (n *TeamNote) Clone() *TeamNote {
	if n == nil {
		return nil
	}
	copy := *n
	copy.Mentions = slices.Clone(n.Mentions)
	slices.Sort(copy.Mentions)
	copy.Mentions = slices.Compact(copy.Mentions)
	if n.Work != nil {
		work := *n.Work
		work.Paths = slices.Clone(n.Work.Paths)
		copy.Work = &work
	}
	return &copy
}

func (n *TeamNote) Check(kind string) error {
	if n == nil {
		return nil
	}
	if kind != "team_discussion" && kind != "team_work" && kind != "team_progress" && kind != "team_message" {
		return faults.Errorf(diagcatalog.RecordCollaborationMetadataRequiresAPublicPeerEvent)
	}
	if n.ReplyTo < 0 || n.BasedOn < 0 || (n.ReplyTo == 0) != (n.ReplyAuthor == "") {
		return faults.Errorf(diagcatalog.RecordInvalidPublicReplyReference)
	}
	if n.Work != nil && (kind != "team_work" || n.Work.Revision == 0) {
		return faults.Errorf(diagcatalog.RecordWorkMetadataRequiresAVersionedSelfDeclaration)
	}
	return nil
}
