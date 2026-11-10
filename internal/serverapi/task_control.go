// Purpose: Expose authenticated task status, approval decisions and cancellation on every protocol.
// Pattern: HTTP control adapter over existing application commands.
// Usage: Native clients use /v1/tasks/{id} controls independently of observer streams.
// Related: internal/serverapi/commands.go, internal/commands/server/controls.go.
package serverapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func (t *Transport) taskControl(w http.ResponseWriter, r *http.Request) {
	api, release, scopeErr := t.requestAPI(r)
	if scopeErr != nil {
		t.writeControlReply(w, r, t.api.errorReply("", scopeErr))
		return
	}
	defer release()
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/tasks/"), "/")
	if len(parts) == 0 || parts[0] == "" || len(parts[0]) > 128 {
		http.NotFound(w, r)
		return
	}
	cmd := &v1.Command{}
	switch {
	case r.Method == "GET" && len(parts) == 1:
		cmd.Operation = &v1.Command_TaskStatus{TaskStatus: &v1.TaskRequest{TaskId: parts[0]}}
	case r.Method == "GET" && len(parts) == 2 && parts[1] == "approvals":
		if _, err := api.Service.TaskStatus(r.Context(), parts[0]); err != nil {
			t.writeControlReply(w, r, api.errorReply("", err))
			return
		}
		state, err := api.Service.StatePage(r.Context(), "", parts[0], "", 1)
		if err != nil {
			t.writeControlReply(w, r, api.errorReply("", err))
			return
		}
		n := &v1.Notification{Body: &v1.Notification_Snapshot{Snapshot: snapshot(state)}}
		api.sanitize(n)
		_ = taskSender(parts[0], func(n *v1.Notification) error {
			t.writeControlReply(w, r, &v1.Reply{Result: &v1.Reply_State{State: n.GetSnapshot()}})
			return nil
		})(n)
		return
	case r.Method == "POST" && len(parts) == 2 && parts[1] == "cancel":
		cmd.Operation = &v1.Command_Cancel{Cancel: &v1.TaskRequest{TaskId: parts[0]}}
	case r.Method == "POST" && len(parts) == 3 && parts[1] == "approvals":
		defer r.Body.Close()
		allow, valid := readApprovalDecision(w, r)
		if !valid {
			t.writeControlReply(w, r, api.errorReply("", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerApprovalDecision)))
			return
		}
		cmd.Operation = &v1.Command_Approve{Approve: &v1.ApprovalDecision{TaskId: parts[0], ApprovalId: parts[2], Allow: allow}}
	default:
		http.NotFound(w, r)
		return
	}
	reply, _ := api.Call(r.Context(), cmd)
	t.writeControlReply(w, r, reply)
}

func readApprovalDecision(w http.ResponseWriter, r *http.Request) (bool, bool) {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return false, false
	}
	key, err := decoder.Token()
	if err != nil || key != "allow" {
		return false, false
	}
	var allow *bool
	if decoder.Decode(&allow) != nil || allow == nil {
		return false, false
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return false, false
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return false, false
	}
	return *allow, true
}
