// Purpose: Own current team contributions and the latest actual change per canonical path.
// Pattern: Pure state transitions; the room owns synchronization.
// Usage: Team work admission and context projection share one request-scoped State.
// Related: internal/agent/team/contribution/context.go, internal/agent/team/work.go.
package contribution

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

type Scope struct {
	summaryHash, contractHash [32]byte
	progressHash              [32]byte
	Member                    string   `json:"member_name"`
	Summary                   string   `json:"summary"`
	Paths                     []string `json:"paths"`
	Contract                  string   `json:"contract,omitempty"`
	Sequence                  int64    `json:"sequence"`
	Revision                  uint64   `json:"revision"`
	ReplyTo                   int64    `json:"reply_to,omitempty"`
}

// ProgressKey identifies ownership and shared behavior, independently of explanatory wording.
func (s Scope) ProgressKey() [32]byte { return s.progressHash }

type Change struct {
	Member   string
	Sequence int64
}
type State struct {
	scopes         map[string]Scope
	changes        map[string]Change
	revision       uint64
	latest, pruned int64
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
					return Scope{}, fmt.Errorf("%s is working on %s; coordinate a handoff or choose another contribution", other, held)
				}
			}
		}
	}
	value.Revision = s.scopes[value.Member].Revision + 1
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
					return fmt.Errorf("%s owns %s; agree on a handoff before changing it", owner, held)
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
			return fmt.Errorf("%s is outside your current contribution; use set_work with concrete paths before changing it", path)
		}
		if s.latest <= seen {
			continue
		}
		for changed, fact := range s.changes {
			if fact.Member != member && fact.Sequence > seen && overlap(changed, path) {
				return fmt.Errorf("%s changed or may have affected %s after your last decision; read the latest result and inspect the file before regenerating this operation", fact.Member, changed)
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
		s.changes[path] = Change{member, sequence}
	}
}

func (s *State) Scope(member string) Scope {
	value := s.scopes[member]
	value.Paths = slices.Clone(value.Paths)
	return value
}

func (s *State) Matches(value Scope) bool {
	current, exists := s.scopes[value.Member]
	return exists && current.summaryHash == value.summaryHash && current.contractHash == value.contractHash && current.ReplyTo == value.ReplyTo && slices.Equal(current.Paths, value.Paths)
}

// SameCoordination ignores wording and reply references while comparing actual ownership and behavior.
func (s *State) SameCoordination(value Scope) bool {
	current, exists := s.scopes[value.Member]
	return exists && current.progressHash == value.progressHash
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
