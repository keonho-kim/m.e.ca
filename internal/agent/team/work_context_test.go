// Purpose: Verify coherent reusable work snapshots, bounded projections and complete underlying declarations.
// Pattern: State transition and concurrent projection tests.
// Usage: Team tests exercise projection invalidation without model or disk I/O.
// Related: internal/agent/team/work_context.go, internal/agent/team/contribution/context.go.
package team

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

func TestOverviewStaysWithin256KiBAndPreservesRegisteredWork(t *testing.T) {
	r := testRoom(t)
	var lastPath string
	var lastMember string
	for i := range 16 {
		name := identity.Name(i)
		if i >= 2 {
			r.reserve(identity.Member{ID: name}, nil)
		}
		paths := make([]string, 64)
		for j := range paths {
			paths[j] = fmt.Sprintf("/project/%d/%s-%d", i, strings.Repeat("가", 110), j)
		}
		scope := contribution.Describe(contribution.Scope{
			Member: name, Summary: strings.Repeat("설명", 600), Contract: strings.Repeat("합의", 600), Paths: paths,
		})
		r.work.Commit(scope, int64(i+1))
		lastPath, lastMember = paths[len(paths)-1], name
	}
	peer := &Peer{ID: "Hana", room: r, idleDirective: "Review the pending work."}
	text := peer.ContextObservations(nil)
	t.Logf("team overview with 16 large declarations: %d bytes", len(text))
	if len(text) > 256*1024 || !utf8.ValidString(text) || !strings.Contains(text, "list_team_members") || !strings.Contains(text, peer.idleDirective) {
		t.Fatalf("overview is oversized or lacks its continuation guidance: %d bytes", len(text))
	}
	last := r.work.Scope(lastMember)
	if len(last.Paths) != 64 || last.Paths[len(last.Paths)-1] != lastPath {
		t.Fatal("overview budget changed the underlying claim")
	}
}

func TestWorkContextTracksMembershipAndReleasesReplacedText(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	p.SetWork("Implement API", []string{"/project/api"}, "", 0)
	first := p.ContextObservations(nil)
	if !strings.Contains(first, "Hana, Ren") || !strings.Contains(first, "Implement API") {
		t.Fatal(first)
	}
	r.reserve(identity.Member{ID: "Arjun"}, nil)
	second := p.ContextObservations(nil)
	if !strings.Contains(second, "Arjun, Hana, Ren") {
		t.Fatal("membership was stale", second)
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
				if text := r.workSnapshot().view.fullOverview(); !strings.Contains(text, "Hana, Ren") {
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

func TestStatusSnapshotOnlyFormatsOverviewOnDemand(t *testing.T) {
	r := testRoom(t)
	peer := &Peer{ID: "Hana", room: r}
	if _, err := peer.SetWork("Inspect component", []string{"/project/component"}, "", 0); err != nil {
		t.Fatal(err)
	}
	first := r.workSnapshot().view
	if first.text != "" {
		t.Fatal("status snapshot rendered an unused overview")
	}
	r.mu.Lock()
	r.latestStatus++
	r.active["Ren"].status = commandStatus{from: "Ren", text: "Running checks", sequence: r.latestStatus}
	r.mu.Unlock()
	next := r.workSnapshot().view
	if next.base != first.base || next.text != "" {
		t.Fatal("status-only change rebuilt stable work or rendered an unused overview")
	}
	if text := next.fullOverview(); !strings.Contains(text, "Running checks") || !strings.Contains(text, "Inspect component") {
		t.Fatal(text)
	}
}

func TestUnrelatedIdlePeerDoesNotHoldReviewedChanges(t *testing.T) {
	r := testRoom(t)
	peer := &Peer{ID: "Hana", room: r}
	other := &Peer{ID: "Ren", room: r}
	if _, err := peer.SetWork("Inspect component", []string{"/project/component"}, "", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := other.SetWork("Edit docs", []string{"/project/docs"}, "", 0); err != nil {
		t.Fatal(err)
	}
	r.mu.Lock()
	r.active["Ren"].waiting = true
	r.mu.Unlock()
	observation, err := r.publish(channel.Message{From: "Hana", Text: "Changed component", Observation: true, ChangedPaths: []string{"/project/component/file.go"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := peer.Inbox(r.ctx); err != nil {
		t.Fatal(err)
	}
	peer.BeginDecision()
	if err := r.work.Check("Ren", []string{"/project/docs/file.md"}, 0); err == nil {
		t.Fatal("a stale decision bypassed the fence for reclaimed changes")
	}
	if err := r.work.Check("Ren", []string{"/project/docs/file.md"}, observation.Sequence); err != nil {
		t.Fatal(err)
	}
}
