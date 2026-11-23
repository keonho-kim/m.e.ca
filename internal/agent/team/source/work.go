// Purpose: Identify a work domain across peer claims and invitations.
// Pattern: Canonical domain key with an optional public proposal reference.
// Usage: Collaboration tools attach the same work identity to active claims and durable notes.
// Related: internal/agent/team/source/public.go, internal/agent/team/contribution/state.go.
package source

import (
	"strconv"
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const (
	MaxWorkDomainCharacters = 128
	MaxWorkDomains          = 64
	MaxWorkRef              = 1<<53 - 1
)

type WorkDomain struct {
	Domain  string `json:"domain"`
	WorkRef int64  `json:"work_ref,omitempty"`
}

func ParseWorkDomain(value WorkDomain) (WorkDomain, error) {
	value.Domain = strings.ToLower(strings.TrimSpace(value.Domain))
	if value.Domain == "" || !utf8.ValidString(value.Domain) || utf8.RuneCountInString(value.Domain) > MaxWorkDomainCharacters || strings.ContainsAny(value.Domain, "\x00\r\n") {
		return WorkDomain{}, faults.Errorf(diagcatalog.ToolRequiresShortLine, "domain", MaxWorkDomainCharacters)
	}
	if value.WorkRef < 0 || value.WorkRef > MaxWorkRef {
		return WorkDomain{}, faults.Errorf(diagcatalog.TeamWorkReferenceMustIdentifyPublicProposal)
	}
	return value, nil
}

func (w WorkDomain) Matches(other WorkDomain) bool {
	return w.Domain != "" && (w.Domain == other.Domain || w.WorkRef != 0 && w.WorkRef == other.WorkRef)
}

func (w WorkDomain) String() string {
	if w.WorkRef != 0 {
		return w.Domain + " (#" + strconv.FormatInt(w.WorkRef, 10) + ")"
	}
	return w.Domain
}
