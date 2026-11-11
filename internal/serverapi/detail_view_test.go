// Purpose: Verify separately paged tool request and result documents never share content or cursor revisions.
// Pattern: Authenticated native API contract tests.
// Usage: go test -race ./internal/serverapi -run TestToolDetailViews.
// Related: internal/serverapi/detail.go, api/meca/v1/control.proto.
package serverapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func TestToolDetailViewsSeparateRequestsAndResults(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	state, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"command": "printf request-sentinel", "content": strings.Repeat("original request\n", 6000)})
	call := tool.Call{ID: "tool-call", Name: "execute_command", Arguments: args}
	if err := f.store.RecordEvent(t.Context(), state.Session.ID, record.Event{Kind: "tool_succeeded", Call: &call, CallID: call.ID, Output: json.RawMessage(`{"stdout":"result-sentinel"}`)}); err != nil {
		t.Fatal(err)
	}
	wire := nativeTransport(t, f, "sse")
	request := &v1.DetailRequest{SessionId: state.Session.ID, Source: &v1.DetailRequest_Tool{Tool: &v1.EventReference{SessionId: state.Session.ID, Sequence: 1}}, View: v1.DetailRequest_TOOL_REQUEST, Limit: 65536}
	first := nativeCommand(t, wire, &v1.Command{Operation: &v1.Command_Detail{Detail: request}}).GetDetail()
	if first == nil || !strings.Contains(first.Text, "request-sentinel") || strings.Contains(first.Text, "result-sentinel") || first.NextOffset >= first.TotalBytes {
		t.Fatalf("request projection/page: %+v", first)
	}
	request.View = v1.DetailRequest_TOOL_RESULT
	output := nativeCommand(t, wire, &v1.Command{Operation: &v1.Command_Detail{Detail: request}}).GetDetail()
	if output == nil || output.Text != `{"stdout":"result-sentinel"}` || output.Revision == first.Revision {
		t.Fatalf("result mixed with request: %+v", output)
	}
	request.Offset = 1
	request.Revision = first.Revision
	reply := nativeCommand(t, wire, &v1.Command{Operation: &v1.Command_Detail{Detail: request}})
	if reply.GetError().GetCode() != "conflict" {
		t.Fatalf("cross-view cursor accepted: %+v", reply)
	}
}

func TestDetailNarrativeViewsAndSourceContracts(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	state, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.RecordEvent(t.Context(), state.Session.ID, record.Event{Kind: "thinking", Text: "response-only", Thinking: "추론-only"}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.AppendMessage(t.Context(), state.Session.ID, model.Message{Role: "assistant", Content: "saved-response"}); err != nil {
		t.Fatal(err)
	}
	wire := nativeTransport(t, f, "sse")
	for _, test := range []struct {
		name     string
		view     v1.DetailRequest_View
		sequence int64
		want     string
	}{
		{"event text", v1.DetailRequest_TEXT, 1, "response-only"},
		{"thinking", v1.DetailRequest_THINKING, 1, "추론-only"},
		{"saved message", v1.DetailRequest_TEXT, 2, "saved-response"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := &v1.DetailRequest{SessionId: state.Session.ID, Source: &v1.DetailRequest_HistorySequence{HistorySequence: test.sequence}, View: test.view}
			page := nativeCommand(t, wire, &v1.Command{Operation: &v1.Command_Detail{Detail: request}}).GetDetail()
			if page == nil || page.Text != test.want || page.ContentType != "text/plain; charset=utf-8" || page.TotalBytes != int64(len(test.want)) {
				t.Fatalf("wrong narrative view: %+v", page)
			}
		})
	}
	for _, request := range []*v1.DetailRequest{
		{View: v1.DetailRequest_TOOL_REQUEST, Source: &v1.DetailRequest_HistorySequence{HistorySequence: 1}},
		{View: v1.DetailRequest_TEXT, Source: &v1.DetailRequest_Tool{Tool: &v1.EventReference{Sequence: 1}}},
		{View: v1.DetailRequest_THINKING, Source: &v1.DetailRequest_ApprovalId{ApprovalId: "not-a-narrative"}},
		{View: v1.DetailRequest_View(100), Source: &v1.DetailRequest_HistorySequence{HistorySequence: 1}},
	} {
		request.SessionId = state.Session.ID
		reply := nativeCommand(t, wire, &v1.Command{Operation: &v1.Command_Detail{Detail: request}})
		if reply.GetError().GetCode() != "invalid_argument" {
			t.Fatalf("incompatible view/source accepted: %+v", reply)
		}
	}
}
