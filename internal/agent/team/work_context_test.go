// Purpose: Verify coherent reusable work snapshots, bounded projections and complete underlying declarations.
// Pattern: State transition and concurrent projection tests.
// Usage: Team tests exercise projection invalidation without model or disk I/O.
// Related: internal/agent/team/work_context.go, internal/agent/team/contribution/context.go.
package team

import (
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

func TestWorkContextTracksMembershipAndReleasesReplacedText(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	p.SetWork("Implement API", []string{"/project/api"}, "", 0)
	first := p.ContextObservations(nil)
	if !strings.Contains(first, "Available places: 6") || !strings.Contains(first, "Implement API") {
		t.Fatal(first)
	}
	r.reserve(identity.Member{ID: "Arjun"})
	second := p.ContextObservations(nil)
	if !strings.Contains(second, "Arjun, Hana, Ren") || !strings.Contains(second, "Available places: 5") {
		t.Fatal("membership and capacity were stale", second)
	}
	if _, err := p.SetWork("Handoff complete", nil, "", 0); err != nil {
		t.Fatal(err)
	}
	if r.workView != nil || strings.Contains(p.ContextObservations(nil), "Implement API") || !strings.Contains(first, "Implement API") {
		t.Fatal("released work retained its cached text or mutated a previous request")
	}
}

func TestWorkContextBoundsRepeatedTextWithoutTruncatingStoredScope(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	summary := strings.Repeat("현재 작업의 자세한 근거. ", 100000)
	paths := make([]string, 64)
	for i := range paths {
		paths[i] = "/project/" + strings.Repeat("a", 4000) + string(rune('A'+i))
	}
	if _, err := p.SetWork(summary, paths, "", 0); err != nil {
		t.Fatal(err)
	}
	text := p.ContextObservations(nil)
	if len(text) > 32*1024 || !utf8.ValidString(text) || !strings.Contains(text, "more paths") {
		t.Fatal("large declarations exceeded the repeated-context budget", len(text))
	}
	full := r.work.Scope(p.ID)
	if len(full.Summary) > 4096 || len(full.Paths) != len(paths) {
		t.Fatal("current state retained unbounded text or lost ownership")
	}
	public, err := p.ReadMessages(r.ctx, 0)
	if err != nil || len(public) != 1 || public[0].Text != summary {
		t.Fatal("full public declaration was lost", err)
	}
	if err := p.CheckWork([]string{paths[len(paths)-1]}); err != nil {
		t.Fatal("omitted presentation path lost its real ownership", err)
	}
}

func TestConcurrentWorkProjectionAndContributionChanges(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	var workers sync.WaitGroup
	for range 3 {
		workers.Go(func() {
			for range 40 {
				if text := r.workSnapshot().view.text; !strings.Contains(text, "Hana, Ren") {
					t.Error("mixed membership snapshot", text)
				}
			}
		})
	}
	for range 40 {
		if _, err := p.SetWork("API", []string{"/project/api"}, "", 0); err != nil {
			t.Fatal(err)
		}
		if _, err := p.SetWork("Finished", nil, "", 0); err != nil {
			t.Fatal(err)
		}
	}
	workers.Wait()
}
