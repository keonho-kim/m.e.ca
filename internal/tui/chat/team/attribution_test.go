// Purpose: Verify public speech remains attached to its sender across empty and interleaved snapshots.
// Pattern: Deterministic presentation regression tests.
// Usage: go test ./internal/tui/chat/team -run 'TestSpeech|TestEmpty'.
// Related: internal/tui/chat/team/events.go, internal/tui/chat/team/state.go.
package team

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestSpeechStartsWithWhitespaceWithoutBorrowingAnotherBlock(t *testing.T) {
	for _, prefix := range []string{"\n", " \n\t", "\n\n"} {
		t.Run(prefix, func(t *testing.T) {
			var state State
			output := viewport.View{Follow: true}
			for _, event := range []runrecord.Event{
				{Kind: "stream", Agent: "Hana", Text: "A contribution"},
				{Kind: "stream", Agent: "Ren", Text: prefix},
				{Kind: "stream", Agent: "Hana", Text: "A contribution complete"},
				{Kind: "stream", Agent: "Ren", Text: prefix + "B contribution"},
				{Kind: "team_progress", Agent: "Ren", Text: prefix + "B contribution"},
				{Kind: "team_progress", Agent: "Hana", Text: "A contribution complete"},
			} {
				state.Apply(event, formatter(), &output)
				assertSpeechOwners(t, output, map[string]string{"A contribution": "Hana", "B contribution": "Ren"})
			}
			assertSpeechCount(t, output, "A contribution complete", 1)
			assertSpeechCount(t, output, "B contribution", 1)
		})
	}
}

func TestEmptySpeechDoesNotCreateAnOrphanedSender(t *testing.T) {
	var state State
	output := viewport.View{Follow: true}
	for _, event := range []runrecord.Event{
		{Kind: "generation_started", Agent: "Hana"},
		{Kind: "stream", Agent: "Hana", Text: "\n\n"},
		{Kind: "progress", Agent: "Hana", Text: "\n\n"},
		{Kind: "team_progress", Agent: "Hana", Text: " \n\t"},
		{Kind: "team_message", Agent: "Ren", Text: "\n\n"},
	} {
		state.Apply(event, formatter(), &output)
	}
	if len(output.Blocks) != 0 {
		t.Fatalf("empty speech left visible sender blocks: %+v", output.Blocks)
	}
	state.Apply(runrecord.Event{Kind: "tool_started", Agent: "Hana", Call: &toolapi.Call{ID: "tool-1", Name: "execute_command", Arguments: []byte(`{"command":"pwd"}`)}}, formatter(), &output)
	assertSpeechCount(t, output, "**Hana**", 1)
}

func TestSpeechRestoresItsSenderAfterHistoryTrimming(t *testing.T) {
	for _, count := range []int{997, 998} {
		var state State
		output := viewport.View{Follow: true}
		for _, event := range []runrecord.Event{
			{Kind: "stream", Agent: "Hana", Text: "A contribution"},
			{Kind: "stream", Agent: "Ren", Text: "B contribution"},
		} {
			state.Apply(event, formatter(), &output)
		}
		for range count {
			output.Append("Earlier activity", false)
		}
		state.Apply(runrecord.Event{Kind: "stream", Agent: "Hana", Text: "A contribution continued"}, formatter(), &output)
		assertSpeechOwners(t, output, map[string]string{"A contribution": "Hana", "B contribution": "Ren"})
		assertSpeechCount(t, output, "A contribution", 1)
	}
}

func TestSpeechCompletionSurvivesEvictionDuringReplacement(t *testing.T) {
	var state State
	output := viewport.View{Follow: true}
	payload := strings.Repeat("content\n", 1024*1024)
	state.Apply(runrecord.Event{Kind: "stream", Agent: "Hana", Text: "A contribution"}, formatter(), &output)
	state.Apply(runrecord.Event{Kind: "stream", Agent: "Ren", Text: payload + payload}, formatter(), &output)
	state.Apply(runrecord.Event{Kind: "team_progress", Agent: "Hana", Text: "A contribution " + payload, Final: true}, formatter(), &output)
	assertSpeechOwners(t, output, map[string]string{"A contribution": "Hana"})
	assertSpeechCount(t, output, "A contribution", 1)
	assertSpeechCount(t, output, "**Hana** ·", 1)
}

func TestSpeechAndToolsKeepOwnersWithIdenticalCallIDs(t *testing.T) {
	var state State
	output := viewport.View{Follow: true}
	calls := map[string]*toolapi.Call{
		"Hana": {ID: "same-id", Name: "execute_command", Arguments: []byte(`{"command":"printf A-command"}`)},
		"Ren":  {ID: "same-id", Name: "execute_command", Arguments: []byte(`{"command":"printf B-command"}`)},
	}
	for _, kind := range []string{"tool_started", "tool_waiting", "tool_succeeded"} {
		for _, id := range []string{"Hana", "Ren"} {
			state.Apply(runrecord.Event{Kind: kind, Agent: id, Call: calls[id], Text: "completed"}, formatter(), &output)
			assertSpeechOwners(t, output, map[string]string{"A-command": "Hana", "B-command": "Ren"})
		}
		if kind == "tool_waiting" {
			for range 1000 {
				output.Append("Earlier activity", false)
			}
		}
	}
	assertSpeechCount(t, output, "A-command", 1)
	assertSpeechCount(t, output, "B-command", 1)
	if len(state.tools) != 0 {
		t.Fatal("completed cards retained active source identities")
	}
}

func assertSpeechOwners(t *testing.T, output viewport.View, markers map[string]string) {
	t.Helper()
	speaker := ""
	for _, block := range output.Blocks {
		for _, id := range []string{"Hana", "Ren"} {
			if strings.Contains(block.Text, "**"+id+"**") {
				speaker = id
			}
		}
		for marker, want := range markers {
			if strings.Contains(block.Text, marker) && speaker != want {
				t.Fatalf("block %d: %q appears under %q, want %q", block.ID, marker, speaker, want)
			}
		}
	}
}

func assertSpeechCount(t *testing.T, output viewport.View, marker string, want int) {
	t.Helper()
	count := 0
	for _, block := range output.Blocks {
		count += strings.Count(block.Text, marker)
	}
	if count != want {
		t.Fatalf("%q appears %d times, want %d", marker, count, want)
	}
}
