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
	Member  string   `json:"member_name"`
	Summary string   `json:"summary"`
	Paths   []string `json:"paths"`
}
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

// Set replaces a contribution atomically: conflicts leave its existing scope intact.
func (s *State) Set(member, summary string, paths []string) error {
	for other, scope := range s.scopes {
		if other == member {
			continue
		}
		for _, held := range scope.Paths {
			for _, path := range paths {
				if overlap(held, path) {
					return fmt.Errorf("%s is working on %s; coordinate a handoff or choose another contribution", other, held)
				}
			}
		}
	}
	if len(paths) == 0 {
		s.Release(member)
		return nil
	}
	if s.scopes == nil {
		s.scopes = make(map[string]Scope)
	}
	s.scopes[member] = Scope{member, summary, slices.Clone(paths)}
	s.revision++
	return nil
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

func (s *State) Matches(member, summary string, paths []string) bool {
	value := s.scopes[member]
	if len(paths) == 0 {
		return len(value.Paths) == 0
	}
	return value.Summary == summary && slices.Equal(value.Paths, paths)
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
