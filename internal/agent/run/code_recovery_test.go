// Purpose: Deliver bounded unreturned tool outputs without repeating real-world effects.
// Pattern: Runtime integration with a scripted model, SQLite and local file operations.
// Usage: go test ./internal/agent/run -run 'TestCodeUndefined|TestCodeDefined|TestCodeFallback|TestCodeUncertain'.
// Related: internal/agent/run/codemode.go, internal/agent/run/code_fallback.go, internal/agent/run/action_result.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

func recoveryRunner(t *testing.T) (Runner, string) {
	t.Helper()
	r, _, project, _, _ := realBatchRunner(t, nil)
	sessions, err := sqlite.OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sessions.Create(t.Context(), project, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	r.Journal, r.Session, r.ProjectRoot = db, db.Session().ID, project
	return r, project
}

func latestCodeOutput(request modelapi.Request) string {
	for i := len(request.Messages) - 1; i >= 0; i-- {
		if m := request.Messages[i]; m.Role == "tool" && m.ToolName == "codemode" {
			return m.Content
		}
	}
	return ""
}

func TestCodeUndefinedResultDeliversOutputsWithoutRepeatingEffects(t *testing.T) {
	r, project := recoveryRunner(t)
	const observed = "RECOVERED_OUTPUT"
	if err := os.WriteFile(filepath.Join(project, "source.txt"), []byte(observed+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	step, commands, writes := 0, 0, 0
	r.Generator = boundaryModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		step++
		var code string
		switch step {
		case 1:
			code = `await tools.write_file({reason:"Create the requested file",path:"prelude.txt",content:"done"});
				const result = await tools.execute_command({
				reason:"Perform the requested effect",
				command:"printf 'once\\n' >> counter.txt; cat source.txt"
			});`
		case 2:
			body := latestCodeOutput(request)
			var response struct {
				Status       string `json:"status"`
				ReturnStatus string `json:"return_status"`
				Fallback     struct {
					Outputs   []codeObservation
					Truncated bool
				}
			}
			if err := json.Unmarshal([]byte(body), &response); err != nil {
				return err
			}
			if response.Status != "completed" || response.ReturnStatus != "undefined" || len(response.Fallback.Outputs) != 2 || response.Fallback.Truncated {
				return fmt.Errorf("missing completed outputs: %s", body)
			}
			if response.Fallback.Outputs[0].Tool != "write_file" || response.Fallback.Outputs[1].Tool != "execute_command" {
				return fmt.Errorf("wrong completed operations: %+v", response.Fallback.Outputs)
			}
			var output struct{ Stdout string }
			if err := json.Unmarshal(response.Fallback.Outputs[1].Value, &output); err != nil {
				return err
			}
			if output.Stdout != observed+"\n" {
				return fmt.Errorf("wrong recovered output: %q", output.Stdout)
			}
			return emit(modelapi.Event{Delta: "Observed " + observed, Done: true})
		default:
			return fmt.Errorf("unnecessary decision %d", step)
		}
		args, _ := json.Marshal(map[string]string{"code": code})
		return emit(modelapi.Event{Calls: []toolapi.Call{{ID: fmt.Sprintf("code_%d", step), Name: "codemode", Arguments: args}}, Done: true})
	})
	result, err := r.Run(t.Context(), "Run the requested effect once and report its output.", approve, func(event runrecord.Event) error {
		if event.Kind == "tool_started" && event.Call != nil && event.Call.Name == "execute_command" {
			commands++
		}
		if event.Kind == "tool_started" && event.Call != nil && event.Call.Name == "write_file" {
			writes++
		}
		return nil
	})
	if err != nil || result.Status != runrecord.Answered || commands != 1 || writes != 1 || step != 2 {
		t.Fatalf("result=%+v commands=%d writes=%d steps=%d err=%v", result, commands, writes, step, err)
	}
	body, err := os.ReadFile(filepath.Join(project, "counter.txt"))
	if err != nil || string(body) != "once\n" {
		t.Fatalf("effect was not preserved exactly once: %q %v", body, err)
	}
}

func TestCodeDefinedNullRemainsAnIntentionalResult(t *testing.T) {
	g := &generator{replies: []string{codeCall(`await tools.inspect_fixture({path:"a"}); return null;`), "Done"}}
	e := &executor{outcomes: []toolapi.Result{{Content: json.RawMessage(`{"private":"unneeded original"}`)}}}
	r := fixture(g, e, &journal{})
	if _, err := r.Run(t.Context(), "Perform the requested operation.", approve, emit); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Status       string
		ReturnStatus string `json:"return_status"`
		Result       json.RawMessage
		Fallback     json.RawMessage
	}
	body := latestCodeOutput(g.requests[1])
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.ReturnStatus != "value" || string(result.Result) != "null" || len(result.Fallback) != 0 {
		t.Fatalf("explicit null lost its meaning: %s", body)
	}
	if strings.Contains(body, "unneeded original") {
		t.Fatal("intentional null exposed intermediate data")
	}
}

type uncertainResultCommit struct{ *sqlite.Store }

func (j uncertainResultCommit) Commit(ctx context.Context, session string, records []runrecord.Entry) error {
	if err := j.Store.Commit(ctx, session, records); err != nil {
		return err
	}
	for _, entry := range records {
		if e := entry.Event; e != nil && e.Kind == "tool_succeeded" && e.Call != nil && e.Call.Name == "execute_command" {
			return fmt.Errorf("simulated uncertain commit acknowledgement")
		}
	}
	return nil
}

func TestCodeUncertainCommitStopsBeforeFallbackDelivery(t *testing.T) {
	r, project := recoveryRunner(t)
	db := r.Journal.(*sqlite.Store)
	r.Journal = uncertainResultCommit{db}
	g := &generator{replies: []string{codeCall(`await tools.execute_command({
		reason:"Perform the requested effect",command:"printf 'once\\n' >> counter.txt"
	});`)}}
	r.Generator = g
	_, err := r.Run(t.Context(), "Run the effect once.", approve, emit)
	if !errors.Is(err, errRuntimeRecord) || len(g.requests) != 1 {
		t.Fatalf("uncertain persistence continued model decisions: requests=%d err=%v", len(g.requests), err)
	}
	body, err := os.ReadFile(filepath.Join(project, "counter.txt"))
	if err != nil || string(body) != "once\n" {
		t.Fatalf("effect was repeated: %q %v", body, err)
	}
	messages, err := db.ContextMessages(t.Context(), r.Session, 0, 1<<20, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Status   string
		Fallback json.RawMessage
	}
	if err := json.Unmarshal([]byte(latestCodeOutput(modelapi.Request{Messages: messages})), &output); err != nil {
		t.Fatal(err)
	}
	if output.Status != "error" || len(output.Fallback) != 0 {
		t.Fatalf("uncertain commit was presented as recoverable success: %+v", output)
	}
}

func TestCodeFallbackBoundsAndRedactsLargeOutputs(t *testing.T) {
	const secret = "PRIVATE_TEST_TOKEN"
	original, err := json.Marshal(map[string]string{"text": secret + strings.Repeat("한글<>\n", 32<<10)})
	if err != nil {
		t.Fatal(err)
	}
	g := &generator{replies: []string{codeCall(`
		await tools.inspect_fixture({path:"a"});
		await tools.inspect_fixture({path:"b"});
	`), "Done"}}
	e := &executor{outcomes: []toolapi.Result{{Content: original}, {Content: original}}}
	r := fixture(g, e, &journal{})
	r.Redact = func(text string) string { return strings.ReplaceAll(text, secret, "[redacted]") }
	if _, err := r.Run(t.Context(), "Read both requested sources.", approve, emit); err != nil {
		t.Fatal(err)
	}
	body := latestCodeOutput(g.requests[1])
	var result struct {
		Fallback struct {
			Outputs   json.RawMessage
			Truncated bool
		}
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Fallback.Truncated || len(result.Fallback.Outputs) > 512<<10 || !utf8.ValidString(body) {
		t.Fatalf("large results were not bounded: truncated=%t bytes=%d", result.Fallback.Truncated, len(result.Fallback.Outputs))
	}
	if strings.Contains(body, secret) || !strings.Contains(body, "[redacted]") || !strings.Contains(body, "한글") {
		t.Fatal("bounded output lost readable text or exposed a secret")
	}
	var outputs []struct {
		Value struct {
			Excerpt   string
			Truncated bool
		}
	}
	if err := json.Unmarshal(result.Fallback.Outputs, &outputs); err != nil || len(outputs) == 0 || !outputs[0].Value.Truncated || !utf8.ValidString(outputs[0].Value.Excerpt) {
		t.Fatalf("truncated output was not identified as a UTF-8 excerpt: %v", err)
	}
	if !strings.Contains(string(original), secret) || e.executed != 2 {
		t.Fatal("fallback mutated the original output or repeated an operation")
	}
}

func TestCodeFallbackDeliversALargeCompleteResultToTheModel(t *testing.T) {
	text := strings.Repeat("Complete output. ", 16<<10) + "END_OF_RESULT"
	original, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		t.Fatal(err)
	}
	g := &generator{replies: []string{codeCall(`await tools.inspect_fixture({path:"a"});`), "Done"}}
	e := &executor{outcomes: []toolapi.Result{{Content: original}}}
	r := fixture(g, e, &journal{})
	if _, err := r.Run(t.Context(), "Read the requested source.", approve, emit); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Fallback struct {
			Outputs   []struct{ Value struct{ Text string } }
			Truncated bool
		}
	}
	if err := json.Unmarshal([]byte(latestCodeOutput(g.requests[1])), &response); err != nil {
		t.Fatal(err)
	}
	if response.Fallback.Truncated || len(response.Fallback.Outputs) != 1 || response.Fallback.Outputs[0].Value.Text != text || e.executed != 1 {
		t.Fatal("model did not receive the complete output from one execution")
	}
}
