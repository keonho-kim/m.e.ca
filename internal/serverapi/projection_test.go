// Purpose: Keep shared replay events immutable when separate subscribers sanitize their wire messages.
// Pattern: Concurrent projection ownership tests.
// Usage: go test -race ./internal/serverapi -run TestProjection.
// Related: internal/serverapi/projection.go, internal/serverapi/watch.go.
package serverapi

import (
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func TestProjectionMasksCredentialAtTruncatedBoundary(t *testing.T) {
	vault := &credentials.Vault{}
	if err := vault.Remember("private-key"); err != nil {
		t.Fatal(err)
	}
	a := &API{Redact: vault.Redact, RedactPreview: func(text string) string { return vault.NewStreamMask()(text, true) }}
	item := &v1.HistoryItem{Entry: &v1.HistoryItem_Message{Message: &v1.TranscriptMessage{Text: "text private-ke"}}, Truncated: true}
	a.sanitize(item)
	if item.GetMessage().Text != "text [redacted]" {
		t.Fatal("credential prefix leaked at preview boundary", item.GetMessage().Text)
	}
}

func TestProjectionDoesNotModifyReplayEvents(t *testing.T) {
	vault := &credentials.Vault{}
	vault.Remember("private-key")
	a := &API{Redact: vault.Redact}
	shared := server.Event{Record: record.Event{Kind: "team_work", Team: &record.TeamNote{Mentions: []string{"private-key"}, Work: &record.TeamWork{Paths: []string{"private-key/file"}}}}}
	projected := event(shared)
	a.sanitize(projected)
	if projected.GetWork().Details.Mentions[0] != "[redacted]" || projected.GetWork().Details.Paths[0] != "[redacted]/file" {
		t.Fatal(projected)
	}
	if shared.Record.Team.Mentions[0] != "private-key" || shared.Record.Team.Work.Paths[0] != "private-key/file" {
		t.Fatal("subscriber modified the shared replay event")
	}
}

func TestProjectionConcurrentSubscribers(t *testing.T) {
	vault := &credentials.Vault{}
	a := &API{Redact: vault.Redact}
	shared := server.Event{Record: record.Event{Kind: "team_work", Team: &record.TeamNote{Mentions: []string{"Hana"}, Work: &record.TeamWork{Paths: []string{"file.go"}}}}}
	effort := "high"
	settings := server.Snapshot{Selection: connection.Selection{Model: "fixture", ReasoningEffort: &effort}}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			<-start
			for range 100 {
				a.sanitize(event(shared))
				a.sanitize(snapshot(settings))
			}
		})
	}
	close(start)
	workers.Wait()
}

func TestProjectionDoesNotModifySelectedModel(t *testing.T) {
	vault := &credentials.Vault{}
	vault.Remember("private-key")
	a := &API{Redact: vault.Redact}
	effort := "private-key"
	shared := server.Snapshot{Selection: connection.Selection{Model: "fixture", ReasoningEffort: &effort}}
	projected := snapshot(shared)
	a.sanitize(projected)
	if projected.Model.GetReasoningEffort() != "[redacted]" {
		t.Fatal(projected)
	}
	if effort != "private-key" {
		t.Fatal("subscriber modified the selected model")
	}
}

func TestProjectionHistoryBoundsToolArguments(t *testing.T) {
	args := []byte(`{"command":"` + strings.Repeat("界", 128<<10) + `"}`)
	item := historyItem(history.Record{Kind: "event", Event: record.Event{Kind: "tool_succeeded", Call: &tool.Call{ID: "command", Name: "execute_command", Arguments: args}}})
	projected := item.GetEvent().GetTool().GetArgumentsJson()
	if !item.GetEvent().Truncated || len(projected) > 16<<10 || !utf8.Valid(projected) {
		t.Fatal("invalid bounded argument preview")
	}
	args[0] = '!'
	if projected[0] != '{' {
		t.Fatal("history projection retained borrowed arguments")
	}
}

func TestProjectionHistoryMarksTruncatedTeamPaths(t *testing.T) {
	paths := make([]string, 33)
	for i := range paths {
		paths[i] = "file.go"
	}
	item := historyItem(history.Record{Kind: "event", Event: record.Event{Kind: "team_work", Team: &record.TeamNote{Work: &record.TeamWork{Paths: paths}}}})
	if !item.GetEvent().Truncated || len(item.GetEvent().GetWork().Details.Paths) != 32 {
		t.Fatal(item)
	}
}
