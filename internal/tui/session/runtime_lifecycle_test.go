// Purpose: Preserve durable terminal events after cancellation and scope transient retry notices.
// Pattern: Deterministic request-delivery and virtual-screen regressions.
// Usage: go test ./internal/tui/session -run 'TestCanceledTeam|TestRecoveredPeer'.
// Related: internal/tui/session/request_delivery.go, internal/tui/session/team_events.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"errors"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestRecoveredPeerClearsOnlyItsRetryNotice(t *testing.T) {
	a := screenFixture(t)
	a.team.enabled = true
	a.team.begin()
	a.receiveRuntime(&runrecord.Event{Kind: "retry", Agent: "Hana", Text: "Provider retry 2/3"})
	a.receiveRuntime(&runrecord.Event{Kind: "stream", Agent: "Ren", Text: "Other peer continues"})
	if !strings.Contains(a.status, "Provider retry") {
		t.Fatal("another peer cleared the retry notice")
	}
	a.receiveRuntime(&runrecord.Event{Kind: "generation_started", Agent: "Hana"})
	a.receiveRuntime(&runrecord.Event{Kind: "stream", Agent: "Hana", Text: "Recovered successfully"})
	if strings.Contains(a.status, "Provider retry") {
		t.Fatal("recovered peer retained its retry notice")
	}
	a.receiveRuntime(&runrecord.Event{Kind: "retry", Agent: "Hana", Text: "Provider retry 3/3"})
	a.status = "Another user notice"
	a.receiveRuntime(&runrecord.Event{Kind: "stream", Agent: "Hana", Text: "Recovered again"})
	if a.status != "Another user notice" || a.chat.retry.text != "" {
		t.Fatal("recovery erased another notice or retained its old retry text")
	}
}

func TestReasoningUsageClearsOnlyRecoveredSourcesRetry(t *testing.T) {
	a := screenFixture(t)
	a.team.enabled = true
	a.team.begin()
	a.receiveRuntime(&runrecord.Event{Kind: "retry", Agent: "Hana", Text: "Provider retry 2/3"})
	for _, e := range []runrecord.Event{
		{Kind: "generation_started", Agent: "Hana"},
		{Kind: "usage_progress", Agent: "Ren", Usage: &modelapi.TokenUsage{Output: 20}},
		{Kind: "usage_progress", Agent: "Hana", Usage: &modelapi.TokenUsage{Input: 100}},
	} {
		a.receiveRuntime(&e)
		if !strings.Contains(a.status, "Provider retry") {
			t.Fatal("retry cleared before its source produced output")
		}
	}
	a.receiveRuntime(&runrecord.Event{Kind: "usage_progress", Agent: "Hana", Usage: &modelapi.TokenUsage{Input: 100, Output: 20}})
	if a.status != "" || a.chat.retry.text != "" {
		t.Fatal("reasoning output retained the retry notice")
	}
	// Providers that report usage only at completion also confirm recovery.
	a.receiveRuntime(&runrecord.Event{Kind: "retry", Text: "Provider retry 2/3"})
	a.status = "Another user notice"
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 1}})
	if a.status != "Another user notice" || a.chat.retry.text != "" {
		t.Fatal("final usage erased an unrelated notice or retained retry state")
	}
}

func TestCanceledRequestDoesNotAdmitFreshWork(t *testing.T) {
	for _, team := range []bool{false, true} {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		for _, kind := range []string{"stream", "tool_started", "tool_waiting", "progress", "retry"} {
			d := requestDelivery{team: team}
			queue := make(chan notice, 1)
			if err := d.emit(ctx, t.Context(), queue, runrecord.Event{Kind: kind, Agent: "Hana", Text: "new work"}); !errors.Is(err, context.Canceled) || len(queue) != 0 {
				t.Fatalf("team=%t: canceled %s reached presentation: %v", team, kind, err)
			}
		}
		for _, kind := range []string{"team_observation", "team_status"} {
			d := requestDelivery{team: team}
			queue := make(chan notice, 1)
			if err := d.emit(ctx, t.Context(), queue, runrecord.Event{Kind: kind}); err != nil || len(queue) != 0 {
				t.Fatal("hidden committed observation caused a delivery failure", err)
			}
		}
	}
}

func TestCanceledTeamStillReceivesDurableTerminalResult(t *testing.T) {
	a := screenFixture(t)
	a.team.enabled = true
	a.team.begin()
	a.work.state = workerAgent
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	delivery := requestDelivery{team: true}
	call := &toolapi.Call{ID: "running", Name: "execute_command", Arguments: []byte(`{"command":"sleep 60"}`)}
	start := runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: call}
	if err := delivery.post(ctx, a.events, notice{Event: &start}); err != nil {
		t.Fatal(err)
	}
	a.receive(<-a.events)
	cancel()
	a.work.state = workerAgentStopping
	finished := runrecord.Event{Kind: "command_finished", Agent: "Hana", Call: call,
		Operation: &toolapi.Operation{Process: &toolapi.ProcessOutcome{CommandID: "cmd_owned", Status: "interrupted"}},
	}
	// The request-delivery boundary chooses the presentation lifetime for durable results.
	if err := delivery.emit(ctx, a.ctx, a.events, finished); err != nil {
		t.Fatal(err)
	}
	if err := delivery.post(a.ctx, a.events, notice{Done: true, Err: context.Canceled}); err != nil {
		t.Fatal(err)
	}
	a.receive(<-a.events)
	a.receive(<-a.events)
	a.render()
	shown := strings.Join(screenRows(a, 24), "\n")
	if a.work.state != workerIdle || strings.Contains(shown, a.language.Text("Running")+"…") || !strings.Contains(shown, a.language.Text("Command interrupted")) {
		t.Fatalf("canceled request kept stale tool state: %s", shown)
	}
}
