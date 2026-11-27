// Purpose: Verify code orchestration uses real approval and durable execution boundaries.
// Pattern: Scripted model integration with observed host effects.
// Usage: go test ./internal/agent/run -run TestCode.
// Related: internal/agent/run/codemode.go, internal/agent/run/code_dispatch.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config/credentials"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func codeCall(code string) string {
	body, _ := json.Marshal(map[string]string{"code": code})
	call, _ := json.Marshal(toolapi.Call{ID: "code", Name: "codemode", Arguments: body})
	return "native:" + string(call)
}

func TestCodeComposesToolsAndRetainsOriginalsOutsideModelContext(t *testing.T) {
	g := &generator{replies: []string{codeCall(`const r = await tools.inspect_fixture({path:"a"}); await tools.write_file({path:"b",content:r.selected}); return {written:"b"};`), "Done"}}
	e := &executor{outcomes: []toolapi.Result{{Content: json.RawMessage(`{"selected":"chosen","unneeded":"private-large-intermediate"}`)}}}
	j := &journal{}
	r := fixture(g, e, j)
	if _, err := r.Run(t.Context(), "compose", approve, emit); err != nil {
		t.Fatal(err)
	}
	if e.executed != 2 || len(g.requests) != 2 {
		t.Fatalf("effects=%d decisions=%d", e.executed, len(g.requests))
	}
	if len(g.requests[0].Tools) != 1 || g.requests[0].Tools[0].Name != "codemode" {
		t.Fatalf("tools=%+v", g.requests[0].Tools)
	}
	for _, m := range g.requests[1].Messages {
		if strings.Contains(m.Content, "private-large-intermediate") {
			t.Fatal("intermediate result entered model context")
		}
		if m.Role == "tool" && m.ToolName != "codemode" {
			t.Fatalf("unpaired inner tool message: %+v", m)
		}
	}
	found := false
	for _, event := range j.events {
		if event.Kind == "tool_succeeded" && event.Call != nil && event.Call.Name == "inspect_fixture" {
			found = strings.Contains(string(event.Output), "private-large-intermediate") && event.Call.ParentExecutionID != ""
		}
	}
	if !found {
		t.Fatal("original nested result was not recorded")
	}
}

func TestCodeApprovalOnlyAppliesToActualToolsAndDenialStopsTheScript(t *testing.T) {
	g := &generator{replies: []string{codeCall(`try { await tools.write_file({path:"a",content:"x"}); } catch {} await tools.write_file({path:"b",content:"y"});`)}}
	e, j := &executor{approval: true}, &journal{}
	r := fixture(g, e, j)
	approvals := 0
	_, err := r.Run(t.Context(), "write", func(context.Context, toolapi.Approval) (bool, error) { approvals++; return false, nil }, emit)
	if !errors.Is(err, errApprovalDenied) || approvals != 1 || e.executed != 0 || e.prepared != 1 {
		t.Fatalf("approvals=%d effects=%d prepared=%d err=%v", approvals, e.executed, e.prepared, err)
	}
}

func TestCodeStopsAfterUncertainResultCommit(t *testing.T) {
	g := &generator{replies: []string{codeCall(`await tools.write_file({path:"a",content:"x"}); await tools.write_file({path:"b",content:"y"});`)}}
	e, j := &executor{}, &journal{failKind: "tool_succeeded"}
	r := fixture(g, e, j)
	_, err := r.Run(t.Context(), "write", approve, emit)
	if !errors.Is(err, errRuntimeRecord) || e.executed != 1 {
		t.Fatalf("effects=%d err=%v", e.executed, err)
	}
}

func TestCodeSyntaxFailureCanBeCorrectedWithoutEffects(t *testing.T) {
	g := &generator{replies: []string{codeCall(`const = ;`), codeCall(`return await tools.inspect_fixture({path:"a"});`), "Done"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(t.Context(), "read", approve, emit)
	if err != nil || result.Status != runrecord.Answered || e.executed != 1 {
		t.Fatalf("%+v effects=%d err=%v", result, e.executed, err)
	}
}

func TestCodeCannotPersistOrExecuteSynthesizedCredential(t *testing.T) {
	vault := &credentials.Vault{}
	vault.Remember("provider-secret")
	g := &generator{replies: []string{codeCall(`try { await tools.write_file({path:"a",content:["provider","secret"].join("-")}); } catch {} return true;`)}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Redact = vault.Redact
	if _, err := r.Run(t.Context(), "write", approve, emit); err == nil || e.prepared != 0 {
		t.Fatalf("protected call reached preparation: %d %v", e.prepared, err)
	}
	body, _ := json.Marshal(struct {
		Messages any
		Events   any
	}{j.messages, j.events})
	if strings.Contains(string(body), "provider-secret") {
		t.Fatal("synthesized credential reached the journal")
	}
	g.replies = []string{codeCall(`return {value:["provider","secret"].join("-")};`), "Done"}
	g.requests = nil
	if _, err := r.Run(t.Context(), "return", approve, emit); err != nil {
		t.Fatal(err)
	}
	body, _ = json.Marshal(struct {
		Messages any
		Events   any
	}{j.messages, j.events})
	if strings.Contains(string(body), "provider-secret") {
		t.Fatal("code return leaked a protected credential")
	}
}

func TestCodeReturnsNewInstructionsBeforeFurtherEffects(t *testing.T) {
	doc := instruction.Document{Metadata: instruction.Metadata{ID: "project:guide", Name: "guide", Source: "project", Description: "Instructions"}, Revision: strings.Repeat("a", 64), Body: "Read this instruction before writing."}
	body, _ := json.Marshal(doc)
	g := &generator{replies: []string{codeCall(`await tools.inspect_fixture({path:"guide"}); await tools.write_file({path:"a",content:"premature"});`), "Read the new guidance"}}
	e, j := &executor{outcomes: []toolapi.Result{{Content: body, Skill: &doc}}}, &journal{}
	r := fixture(g, e, j)
	result, err := r.Run(t.Context(), "read instructions", approve, emit)
	if err != nil || result.HadToolFailure || e.executed != 1 {
		t.Fatalf("effects=%d result=%+v err=%v", e.executed, result, err)
	}
	found := false
	for _, message := range g.requests[1].Messages {
		found = found || strings.Contains(message.Content, `"status":"needs_model"`) && strings.Contains(message.Content, doc.Body)
	}
	if !found {
		t.Fatal("new instructions did not reach the model boundary")
	}
}

type deniedCodeTools struct{ *executor }

func (e deniedCodeTools) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	if call.Name == "inspect_fixture" {
		return nil, faults.New(diag.PermissionDenied, diag.CodeUnknownTool, call.Name)
	}
	return e.executor.Prepare(ctx, settings, call)
}

func TestCodeCannotCatchAPolicyDenialAndContinueEffects(t *testing.T) {
	g := &generator{replies: []string{codeCall(`try { await tools.inspect_fixture({path:"blocked"}); } catch {} await tools.write_file({path:"a",content:"not allowed"});`)}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Tools = deniedCodeTools{e}
	_, err := r.Run(t.Context(), "inspect", approve, emit)
	if err == nil || e.prepared != 0 || e.executed != 0 {
		t.Fatalf("prepared=%d effects=%d err=%v", e.prepared, e.executed, err)
	}
}

type canceledCodeJournal struct {
	*journal
	cancel context.CancelFunc
}

func (j *canceledCodeJournal) Commit(ctx context.Context, session string, entries []runrecord.Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := j.journal.Commit(ctx, session, entries); err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Message != nil && len(entry.Message.Calls) > 0 {
			j.cancel()
		}
	}
	return nil
}

func TestCodeCancellationBeforeStartupStillRecordsItsOutcome(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	g := &generator{replies: []string{codeCall(`await tools.write_file({path:"a",content:"must not run"});`)}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Journal = &canceledCodeJournal{journal: j, cancel: cancel}
	_, err := r.Run(ctx, "write", approve, emit)
	if !errors.Is(err, context.Canceled) || e.executed != 0 {
		t.Fatalf("effects=%d err=%v", e.executed, err)
	}
	for _, message := range j.messages {
		if message.Role == "tool" && message.ToolName == "codemode" && strings.Contains(message.Content, `"status":"not_executed"`) {
			return
		}
	}
	t.Fatal("accepted code call lost its canceled outcome")
}

func TestCodeRespectsTheConfiguredArgumentLimit(t *testing.T) {
	g := &generator{replies: []string{codeCall(`return "` + strings.Repeat("x", 2048) + `";`), "The request was too large"}}
	e, j := &executor{}, &journal{}
	r := fixture(g, e, j)
	r.Limits = DefaultLimits()
	r.Limits.MaxArgumentBytes = 1024
	result, err := r.Run(t.Context(), "run", approve, emit)
	if err != nil || !result.HadToolFailure || e.prepared != 0 {
		t.Fatalf("%+v prepared=%d err=%v", result, e.prepared, err)
	}
	for _, event := range j.events {
		if event.Kind == "tool_started" {
			t.Fatal("oversized code started execution")
		}
	}
}
