// Purpose: Own current team contributions and the latest actual change per canonical path.
// Pattern: Pure state transitions; the room owns synchronization.
// Usage: Team work admission and context projection share one request-scoped State.
// Related: internal/agent/team/contribution/context.go, internal/agent/team/work.go.
package contribution

import (
	"crypto/sha256"
	"path/filepath"
	"slices"
	"strings"

	"github.com/keonho-kim/meca/internal/agent/team/source"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Scope struct {
	summaryHash, contractHash [32]byte
	progressHash              [32]byte
	Member                    string              `json:"member_name"`
	Summary                   string              `json:"summary"`
	Paths                     []string            `json:"paths"`
	Contract                  string              `json:"contract,omitempty"`
	Sequence                  int64               `json:"sequence"`
	Revision                  uint64              `json:"revision"`
	ReplyTo                   int64               `json:"reply_to,omitempty"`
	Domains                   []source.WorkDomain `json:"domains,omitzero"`
}

// ProgressKey identifies ownership and shared behavior, independently of explanatory wording.
func (s Scope) ProgressKey() [32]byte {
	if s.Domains == nil {
		return s.progressHash
	}
	hash := sha256.Sum256(s.progressHash[:])
	var pair [64]byte
	for _, domain := range s.Domains {
		key := sha256.Sum256([]byte(domain.String()))
		copy(pair[:32], hash[:])
		copy(pair[32:], key[:])
		hash = sha256.Sum256(pair[:])
	}
	return hash
}

type Change struct {
	Member   string
	Sequence int64
}

const pathFactBytes = 1 << 20

type State struct {
	scopes           map[string]Scope
	changes          map[string]Change
	changeBytes      int
	revision         uint64
	latest, pruned   int64
	discardedThrough int64
}

func contains(parent, path string) bool {
	return parent == path || strings.HasPrefix(path, parent+string(filepath.Separator))
}
func overlap(a, b string) bool { return contains(a, b) || contains(b, a) }

// Prepare validates a described candidate before publication freezes its paths.
func (s *State) Prepare(value Scope) (Scope, error) {
	for other, scope := range s.scopes {
		if other == value.Member {
			continue
		}
		for _, held := range scope.Paths {
			for _, path := range value.Paths {
				if overlap(held, path) {
					return Scope{}, faults.Errorf(diagcatalog.ContributionIsWorkingOnCoordinateAHandoffOr, other, held)
				}
			}
		}
	}
	current := s.scopes[value.Member]
	if value.Domains == nil {
		value.Domains = current.Domains
	}
	value.Revision = current.Revision + 1
	value.Sequence = 0
	return value, nil
}

// Commit installs the frozen candidate after publication. Room serializes prepare/commit pairs.
func (s *State) Commit(value Scope, sequence int64) {
	if s.scopes == nil {
		s.scopes = make(map[string]Scope)
	}
	value.Sequence = sequence
	s.scopes[value.Member] = value
	s.revision++
}

// Check distinguishes missing ownership, conflicting work and unseen actual changes.
func (s *State) Check(member string, paths []string, seen int64) error {
	own := s.scopes[member]
	for _, path := range paths {
		for owner, scope := range s.scopes {
			for _, held := range scope.Paths {
				if owner != member && overlap(held, path) {
					return faults.Errorf(diagcatalog.ContributionOwnsAgreeOnAHandoffBeforeChanging, owner, held)
				}
			}
		}
		covered := false
		for _, held := range own.Paths {
			if contains(held, path) {
				covered = true
				break
			}
		}
		if !covered {
			return faults.Errorf(diagcatalog.ContributionIsOutsideYourCurrentContributionUseSet, path)
		}
		if s.latest <= seen {
			continue
		}
		if seen < s.discardedThrough {
			return faults.Errorf(diagcatalog.ContributionPathFactsNeedNewDecision)
		}
		for changed, fact := range s.changes {
			if fact.Member != member && fact.Sequence > seen && overlap(changed, path) {
				return faults.Errorf(diagcatalog.ContributionChangedOrMayHaveAffectedAfterYour, fact.Member, changed)
			}
		}
	}
	return nil
}

func (s *State) Record(member string, paths []string, sequence int64) {
	if len(paths) > 0 {
		s.latest = max(s.latest, sequence)
	}
	if len(paths) > 0 && s.changes == nil {
		s.changes = make(map[string]Change)
	}
	for _, path := range paths {
		if _, exists := s.changes[path]; !exists {
			cost := len(path) + 64
			if cost > pathFactBytes-s.changeBytes {
				// A coarse revision fence preserves stale-write checks after dropping detail.
				s.discardedThrough = max(s.discardedThrough, s.latest)
				s.changes = nil
				s.changeBytes = 0
				return
			}
			s.changeBytes += cost
		}
		s.changes[path] = Change{member, sequence}
	}
}

func (s *State) Scope(member string) Scope {
	value := s.scopes[member]
	value.Paths = slices.Clone(value.Paths)
	value.Domains = slices.Clone(value.Domains)
	return value
}

// HoldsDomain uses the invitation until the peer explicitly declares its ongoing domains.
func (s *State) HoldsDomain(member string, invitation, work source.WorkDomain) bool {
	domains := s.scopes[member].Domains
	if domains == nil {
		return invitation.Matches(work)
	}
	for _, domain := range domains {
		if domain.Matches(work) {
			return true
		}
	}
	return false
}

func (s *State) DomainsChanged(value Scope) bool {
	domains := s.scopes[value.Member].Domains
	return (domains == nil) != (value.Domains == nil) || !slices.Equal(domains, value.Domains)
}

func (s *State) Matches(value Scope) bool {
	current, exists := s.scopes[value.Member]
	return exists && current.summaryHash == value.summaryHash && current.contractHash == value.contractHash && current.ReplyTo == value.ReplyTo && slices.Equal(current.Paths, value.Paths) && !s.DomainsChanged(value)
}

// SameCoordination ignores wording and reply references while comparing actual ownership and behavior.
func (s *State) SameCoordination(value Scope) bool {
	current, exists := s.scopes[value.Member]
	return exists && current.progressHash == value.progressHash && !s.DomainsChanged(value)
}

// ContractChanged includes removal and compares original text rather than context previews.
func (s *State) ContractChanged(value Scope) bool {
	current, exists := s.scopes[value.Member]
	if !exists {
		return value.Contract != ""
	}
	return current.contractHash != value.contractHash
}

func (s *State) HasPaths(member string, paths []string) bool {
	return slices.Equal(s.scopes[member].Paths, paths)
}

// Related conservatively targets current work; unknown scope or targets remain visible to all.
func (s *State) Related(member, source string, paths []string) bool {
	own := s.scopes[member].Paths
	old := s.scopes[source].Paths
	if len(own) == 0 || len(paths) == 0 && len(old) == 0 {
		return true
	}
	for _, held := range own {
		for _, path := range paths {
			if overlap(held, path) {
				return true
			}
		}
		for _, path := range old {
			if overlap(held, path) {
				return true
			}
		}
	}
	return false
}

func (s *State) Release(member string) {
	if _, exists := s.scopes[member]; !exists {
		return
	}
	delete(s.scopes, member)
	if len(s.scopes) == 0 {
		s.scopes = nil
	}
	s.revision++
}

// Prune releases path facts after every active decision has already seen them.
func (s *State) Prune(through int64) {
	if through <= s.pruned || len(s.changes) == 0 {
		return
	}
	s.pruned = through
	before := len(s.changes)
	for path, fact := range s.changes {
		if fact.Sequence <= through {
			s.discardedThrough = max(s.discardedThrough, fact.Sequence)
			s.changeBytes -= len(path) + 64
			delete(s.changes, path)
		}
	}
	if len(s.changes) == 0 {
		s.changes = nil
	} else if before >= 1024 && len(s.changes) < before/4 {
		// Deleted keys release strings; rebuild a sparse map to release its high-water capacity.
		next := make(map[string]Change, len(s.changes))
		for path, change := range s.changes {
			next[path] = change
		}
		s.changes = next
	}
}

// Revision changes only when the current contribution projection can change.
func (s *State) Revision() uint64 { return s.revision }
