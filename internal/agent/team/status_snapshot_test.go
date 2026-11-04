// Purpose: Keep passive command observations bounded while preserving their public originals and important facts.
// Pattern: Room projection and admission regression tests.
// Usage: Team tests cover repeated commands, failures and full-scope lookup summaries.
// Related: internal/agent/team/command_status.go, internal/agent/team/inbox.go, internal/agent/team/membership.go.
package team

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestPassiveStatusReplacesTransientViewWithoutGrowingPrivateHistory(t *testing.T) {
	r := testRoom(t)
	a, b := &Peer{ID: "Hana", room: r}, &Peer{ID: "Ren", room: r}
	for i := range 40 {
		command := fmt.Sprintf("cat file-%d.txt", i)
		op := &toolapi.Operation{Name: "execute_command", Status: "tool_succeeded", Command: command,
			Process: &toolapi.ProcessOutcome{CommandID: fmt.Sprint(i), Status: "exited"}}
		if err := a.Emit(runrecord.Event{Kind: "tool_succeeded", Operation: op}); err != nil {
			t.Fatal(err)
		}
		batch, err := b.Inbox(r.ctx)
		if err != nil || len(batch.Messages) != 0 || batch.Cursor == 0 {
			t.Fatal("passive status entered permanent peer history", batch, err)
		}
	}
	text := b.ContextObservations(nil)
	if !strings.Contains(text, "file-39.txt") || strings.Contains(text, "file-0.txt") {
		t.Fatal("transient command state accumulated old commands", text)
	}
	values, err := b.ReadMessages(r.ctx, 0)
	if err != nil || len(values) != 40 {
		t.Fatal("original public history was discarded", len(values), err)
	}
	op := &toolapi.Operation{Name: "execute_command", Status: "tool_failed", Failure: &toolapi.Failure{Code: "command_failed", Detail: "check failed"},
		Process: &toolapi.ProcessOutcome{CommandID: "failure", Status: "exited", ExitCode: 1}}
	if err := a.Emit(runrecord.Event{Kind: "tool_failed", Operation: op}); err != nil {
		t.Fatal(err)
	}
	batch, err := b.Inbox(r.ctx)
	if err != nil || len(batch.Messages) != 1 || !strings.Contains(batch.Messages[0].Content, "check failed") || strings.Contains(b.ContextObservations(nil), "file-39.txt") {
		t.Fatal("failure was hidden by a stale successful status", batch, err)
	}
}

func TestMemberOverviewBoundsSummaryAndKeepsAllOwnedPaths(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	summary := strings.Repeat("상세 작업 근거 ", 100000)
	paths := []string{"/project/api", "/project/tests"}
	if _, err := p.SetWork(summary, paths, "", 0); err != nil {
		t.Fatal(err)
	}
	page, err := p.Members(r.ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range page.Members {
		if member.ID == p.ID {
			if member.Work == nil || len(member.Work.Summary) > 5000 || !utf8.ValidString(member.Work.Summary) || len(member.Work.Paths) != 2 {
				t.Fatal("member query copied the full declaration or lost ownership", member)
			}
			if len(r.work.Scope(p.ID).Summary) > 4096 {
				t.Fatal("current scope retained an unbounded declaration")
			}
			return
		}
	}
	t.Fatal("current member was absent from the page")
}
