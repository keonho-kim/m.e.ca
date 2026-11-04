// Purpose: Prevent duplicate contributions and preserve atomic handoffs and actual-change boundaries.
// Pattern: Pure state regression tests.
// Usage: Team tests run without filesystem, model calls or timing dependencies.
// Related: internal/agent/team/contribution/state.go, internal/agent/team/contribution/context.go.
package contribution

import (
	"fmt"
	"strings"
	"testing"
)

func setScope(s *State, member, summary string, paths []string) error {
	value, err := s.Prepare(Describe(Scope{Member: member, Summary: summary, Paths: paths}))
	if err == nil {
		s.Commit(value, int64(value.Revision))
	}
	return err
}

func TestOwnershipConflictIsAtomicAndReleaseAllowsHandoff(t *testing.T) {
	var s State
	if err := setScope(&s, "Hana", "API", []string{"/p/api"}); err != nil {
		t.Fatal(err)
	}
	if err := setScope(&s, "Ren", "tests", []string{"/p/tests"}); err != nil {
		t.Fatal(err)
	}
	if err := setScope(&s, "Ren", "duplicate API", []string{"/p/api/file", "/p/other"}); err == nil || !strings.Contains(err.Error(), "Hana") {
		t.Fatal("overlap accepted", err)
	}
	if err := s.Check("Ren", []string{"/p/tests/test"}, 0); err != nil {
		t.Fatal("conflict replaced existing work", err)
	}
	s.Release("Hana")
	if err := setScope(&s, "Ren", "API handoff", []string{"/p/api"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Check("Ren", []string{"/p/api/file"}, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.Check("Ren", []string{"/p/api-other/file"}, 0); err == nil {
		t.Fatal("path prefix escaped scope")
	}
}

func TestPruningPreservesUnseenFactsAndNewerChanges(t *testing.T) {
	var s State
	setScope(&s, "Ren", "integration", []string{"/p"})
	for i := range 2048 {
		s.Record("Hana", []string{fmt.Sprintf("/p/%d", i)}, 10)
	}
	s.Record("Hana", []string{"/p/current"}, 20)
	s.Prune(10)
	if len(s.changes) != 1 || s.Check("Ren", []string{"/p/current"}, 10) == nil {
		t.Fatal("pruning discarded an unseen effect")
	}
	s.Prune(10)
	s.Record("Hana", []string{"/p/new"}, 21)
	if err := s.Check("Ren", []string{"/p/new"}, 20); err == nil {
		t.Fatal("review fast path ignored a newly recorded effect")
	}
	s.Prune(21)
	if s.changes != nil {
		t.Fatal("fully reviewed changes retained their index")
	}
}

func TestConflictErrorsDoNotRepeatLargeDeclarations(t *testing.T) {
	var s State
	setScope(&s, "Hana", strings.Repeat("Detailed contribution ", 100000), []string{"/p/api"})
	err := setScope(&s, "Ren", "conflicting scope", []string{"/p/api/file"})
	if err == nil || len(err.Error()) > 256 || !strings.Contains(err.Error(), "Hana") {
		t.Fatal("conflict copied the full declaration or omitted its owner", err)
	}
}

func TestActualChangesInvalidateOnlyRelatedOlderCandidates(t *testing.T) {
	var s State
	setScope(&s, "Ren", "integration", []string{"/p/api", "/p/tests"})
	s.Record("Hana", []string{"/p/api/file"}, 10)
	if err := s.Check("Ren", []string{"/p/api/file"}, 9); err == nil {
		t.Fatal("stale candidate accepted")
	}
	if err := s.Check("Ren", []string{"/p/tests/test"}, 9); err != nil {
		t.Fatal("unrelated change blocked work", err)
	}
	if err := s.Check("Ren", []string{"/p/api/file"}, 10); err != nil {
		t.Fatal("reviewed change blocked work", err)
	}
	s.Record("Ren", []string{"/p/api/file"}, 11)
	if err := s.Check("Ren", []string{"/p/api/file"}, 10); err != nil {
		t.Fatal("own batch result invalidated siblings", err)
	}
	before := s.Snapshot()
	s.Release("Ren")
	if len(before) == 0 || len(s.Snapshot()) != 0 {
		t.Fatal("closed work retained in context")
	}
}

func TestConsumedFactsAndReleasedScopesDropReferences(t *testing.T) {
	var s State
	setScope(&s, "Hana", "current work", []string{"/p/file"})
	s.Record("Ren", []string{"/p/file"}, 10)
	s.Record("Ren", []string{"/p/file"}, 11)
	if len(s.changes) != 1 {
		t.Fatal("repeated path changes accumulated history")
	}
	s.Prune(10)
	if len(s.changes) != 1 {
		t.Fatal("unseen change was dropped")
	}
	s.Prune(11)
	setScope(&s, "Hana", "Finished", nil)
	if scope := s.Scope("Hana"); len(scope.Paths) != 0 || scope.Summary != "Finished" {
		t.Fatal("released work retained the previous scope")
	}
	s.Release("Hana")
	if s.changes != nil || s.scopes != nil {
		t.Fatal("consumed work retained its indexes or projection")
	}
}
