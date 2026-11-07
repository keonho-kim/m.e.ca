// Purpose: Verify compatible admission, continuation, retention and approval behavior through HTTP.
// Pattern: Engine integration tests using real storage and controlled model callbacks.
// Usage: go test -race ./internal/serverapi -run TestOpenAI.
// Related: internal/commands/server/submission.go, internal/serverapi/openai/handler.go.
package serverapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/tool"
)

func responseID(t *testing.T, body []byte) string {
	t.Helper()
	var value struct{ ID string }
	if err := json.Unmarshal(body, &value); err != nil || value.ID == "" {
		t.Fatalf("response ID missing: %s (%v)", body, err)
	}
	return value.ID
}

func TestOpenAIIdempotencyAndHistoryIsolation(t *testing.T) {
	var calls atomic.Int32
	requests := make(chan model.Request, 3)
	f := newFixture(t, generateFunc(func(_ context.Context, r model.Request, emit func(model.Event) error) error {
		calls.Add(1)
		requests <- r
		return emit(model.Event{Delta: "done", Done: true})
	}), false)
	transport := openAITransport(t, f, "grpc")
	body := `{"model":"meca","messages":[{"role":"system","content":"client rule"},{"role":"user","content":"old user"},{"role":"assistant","content":"external answer"},{"role":"user","content":"new user"}]}`
	first := openAIRequest(t, transport, "POST", "/v1/chat/completions", body, "same-key")
	if first.Code != 200 {
		t.Fatal(first.Code, first.Body.String())
	}
	firstTask, err := f.app.TaskStatus(t.Context(), first.Header().Get("X-Meca-Task-ID"))
	if err != nil || firstTask.Session == "" || f.app.State().Session.ID != "" {
		t.Fatal("compatible request changed native selection", firstTask, err)
	}
	request := <-requests
	contents := []string{}
	for _, message := range request.Messages {
		contents = append(contents, message.Content)
	}
	joined := strings.Join(contents, "\n")
	if !strings.Contains(joined, "client rule") || !strings.Contains(joined, "old user") || !strings.Contains(joined, "external answer") || request.Messages[len(request.Messages)-1].Content != "new user" {
		t.Fatal(request.Messages)
	}
	second := openAIRequest(t, transport, "POST", "/v1/chat/completions", body, "same-key")
	if second.Code != 200 || first.Body.String() != second.Body.String() || calls.Load() != 1 {
		t.Fatal(second.Code, second.Body.String(), calls.Load())
	}
	conflict := openAIRequest(t, transport, "POST", "/v1/chat/completions", strings.Replace(body, "old user", "changed user", 1), "same-key")
	if conflict.Code != 409 || calls.Load() != 1 {
		t.Fatal(conflict.Code, conflict.Body.String())
	}
	third := openAIRequest(t, transport, "POST", "/v1/chat/completions", `{"model":"meca","messages":[{"role":"user","content":"independent"}]}`, "")
	thirdTask, err := f.app.TaskStatus(t.Context(), third.Header().Get("X-Meca-Task-ID"))
	if third.Code != 200 || err != nil || thirdTask.Session == firstTask.Session || thirdTask.Session == "" || f.app.State().Session.ID != "" || calls.Load() != 2 {
		t.Fatal(third.Code, third.Body.String(), f.app.State())
	}
	request = <-requests
	for _, message := range request.Messages[1:] {
		if strings.Contains(message.Content, "old user") || strings.Contains(message.Content, "external answer") {
			t.Fatal("isolated request inherited history", request.Messages)
		}
	}
}

func TestOpenAIResponseContinuationAndRetention(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := openAITransport(t, f, "socket")
	first := openAIRequest(t, transport, "POST", "/v1/responses", `{"model":"meca","input":"first"}`, "first-response")
	if first.Code != 200 {
		t.Fatal(first.Code, first.Body.String())
	}
	id := responseID(t, first.Body.Bytes())
	firstTask, err := f.app.TaskStatus(t.Context(), first.Header().Get("X-Meca-Task-ID"))
	if err != nil {
		t.Fatal(err)
	}
	lookup := openAIRequest(t, transport, "GET", "/v1/responses/"+id, "", "")
	if lookup.Code != 200 || first.Body.String() != lookup.Body.String() {
		t.Fatal(lookup.Code, lookup.Body.String())
	}
	second := openAIRequest(t, transport, "POST", "/v1/responses", `{"model":"meca","input":"second","previous_response_id":"`+id+`"}`, "second-response")
	secondTask, err := f.app.TaskStatus(t.Context(), second.Header().Get("X-Meca-Task-ID"))
	if second.Code != 200 || err != nil || secondTask.Session != firstTask.Session || firstTask.Session == "" || f.app.State().Session.ID != "" {
		t.Fatal(second.Code, second.Body.String())
	}
	stale := openAIRequest(t, transport, "POST", "/v1/responses", `{"model":"meca","input":"stale","previous_response_id":"`+id+`"}`, "stale-response")
	if stale.Code != 409 {
		t.Fatal(stale.Code, stale.Body.String())
	}
	private := openAIRequest(t, transport, "POST", "/v1/responses", `{"model":"meca","input":"private","store":false}`, "private-response")
	if private.Code != 200 {
		t.Fatal(private.Code, private.Body.String())
	}
	privateID := responseID(t, private.Body.Bytes())
	hidden := openAIRequest(t, transport, "GET", "/v1/responses/"+privateID, "", "")
	if hidden.Code != 404 {
		t.Fatal(hidden.Code, hidden.Body.String())
	}
	if again := openAIRequest(t, transport, "POST", "/v1/responses", `{"model":"meca","input":"private","store":false}`, "private-response"); again.Code != 200 || again.Body.String() != private.Body.String() {
		t.Fatal(again.Code, again.Body.String())
	}
}

func TestOpenAIBackgroundCancelAndBusyAdmission(t *testing.T) {
	started := make(chan struct{})
	var calls atomic.Int32
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, _ func(model.Event) error) error {
		calls.Add(1)
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}), false)
	transport := openAITransport(t, f, "sse")
	body := `{"model":"meca","input":"wait","background":true}`
	first := openAIRequest(t, transport, "POST", "/v1/responses", body, "background-request")
	if first.Code != 200 {
		t.Fatal(first.Code, first.Body.String())
	}
	id := responseID(t, first.Body.Bytes())
	select {
	case <-started:
	case <-deadline(t).Done():
		t.Fatal("model did not start")
	}
	again := openAIRequest(t, transport, "POST", "/v1/responses", body, "background-request")
	if again.Code != 200 || responseID(t, again.Body.Bytes()) != id || calls.Load() != 1 {
		t.Fatal(again.Code, again.Body.String())
	}
	busy := openAIRequest(t, transport, "POST", "/v1/responses", body, "another-request")
	if busy.Code != 409 || calls.Load() != 1 {
		t.Fatal(busy.Code, busy.Body.String())
	}
	cancelled := openAIRequest(t, transport, "POST", "/v1/responses/"+id+"/cancel", "", "")
	if cancelled.Code != 200 || !strings.Contains(cancelled.Body.String(), `"status":"cancelled"`) {
		t.Fatal(cancelled.Code, cancelled.Body.String())
	}
	if twice := openAIRequest(t, transport, "POST", "/v1/responses/"+id+"/cancel", "", ""); twice.Code != 200 || twice.Body.String() != cancelled.Body.String() {
		t.Fatal(twice.Code, twice.Body.String())
	}
}

func TestOpenAIApprovalFailsWithoutHangingOrExecuting(t *testing.T) {
	f := newFixture(t, generateFunc(func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
		return emit(model.Event{Calls: []tool.Call{{ID: "write", Name: "write_file", Arguments: json.RawMessage(`{"path":"never.txt","content":"unapproved"}`)}}, Done: true})
	}), true)
	transport := openAITransport(t, f, "sse")
	reply := openAIRequest(t, transport, "POST", "/v1/chat/completions", `{"model":"meca","messages":[{"role":"user","content":"write"}]}`, "")
	if reply.Code != 409 || !strings.Contains(reply.Body.String(), `"code":"approval_required"`) || f.executed.Load() != 0 {
		t.Fatal(reply.Code, reply.Body.String(), f.executed.Load())
	}
	if len(f.app.State().Approvals) != 0 {
		t.Fatal("approval left pending")
	}
	if f.outputReads.Load() != 0 {
		t.Fatal("failed Chat request loaded discarded output", f.outputReads.Load())
	}
}

func TestOpenAIRejectsUnsupportedFieldsBeforeAdmission(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := openAITransport(t, f, "sse")
	for _, body := range []string{
		`{"model":"meca","input":"hello","tools":[]}`,
		`{"model":"meca","input":"hello","temperature":0.1}`,
		`{"model":"meca","input":"hello","stream":null}`,
		`{"model":"meca","model":"other","input":"hello"}`,
		`{"model":"meca","input":[{"role":"user","content":[{"type":"input_image","image_url":"https://example.com/image"}]}]}`,
		`{"model":"meca","input":"hello","background":true,"store":false}`,
	} {
		reply := openAIRequest(t, transport, "POST", "/v1/responses", body, "")
		if reply.Code != http.StatusBadRequest || !strings.Contains(reply.Body.String(), `"error"`) {
			t.Fatal(reply.Code, reply.Body.String())
		}
	}
	if f.app.State().Task != nil || f.app.State().Session.ID != "" {
		t.Fatal("invalid requests changed engine state", f.app.State())
	}
}
