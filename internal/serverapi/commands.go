// Purpose: Dispatch typed external commands to the server use cases without owning execution state.
// Pattern: Transport boundary adapter.
// Usage: HTTP, WebSocket and Protobuf RPC use the same API.Call implementation.
// Related: api/meca/v1/control.proto, internal/commands/server/service.go.
package serverapi

import (
	"context"
	"errors"
	"sync"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	"github.com/keonho-kim/meca/internal/serverapi/detail"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/storage/capacity"
	"github.com/keonho-kim/meca/internal/storage/history"
)

type API struct {
	Service          *server.Service
	Redact           func(string) string
	RedactPreview    func(string) string
	RedactionVersion func() int
	Details          *detail.Cache
	readOnce         sync.Once
	reads            *admission.Budget
}

func (a *API) Call(ctx context.Context, cmd *v1.Command) (*v1.Reply, error) {
	reply := &v1.Reply{Id: cmd.GetId(), Result: &v1.Reply_Ok{Ok: &v1.EmptyResult{}}}
	if len(cmd.GetId()) > 128 {
		return a.errorReply("", &server.Error{Code: "invalid_argument", Message: "command ID exceeds 128 bytes"}), nil
	}
	var err error
	switch cmd.GetOperation().(type) {
	case *v1.Command_ListSessions, *v1.Command_History, *v1.Command_Detail:
		release, err := a.readLimit().Acquire(ctx, 1)
		if err != nil {
			return a.errorReply(cmd.GetId(), err), nil
		}
		defer release()
	}
	switch op := cmd.GetOperation().(type) {
	case *v1.Command_ListSessions:
		var items v1.Sessions
		limit := int(op.ListSessions.GetLimit())
		if limit == 0 {
			limit = 50
		}
		values, next, cause := a.Service.Sessions(ctx, op.ListSessions.GetCursor(), limit)
		items.NextCursor = next
		err = cause
		for _, value := range values {
			items.Items = append(items.Items, &v1.Session{Id: value.ID, ProjectRoot: value.ProjectRoot, DisplayRoot: value.DisplayRoot, Summary: value.Summary, Updated: value.Updated, Unavailable: value.Unavailable})
		}
		reply.Result = &v1.Reply_Sessions{Sessions: &items}
	case *v1.Command_SelectSession:
		value, cause := a.Service.Select(ctx, op.SelectSession.GetSessionId(), op.SelectSession.GetCreate())
		err = cause
		reply.Result = &v1.Reply_State{State: snapshot(value)}
	case *v1.Command_CloseSession:
		err = a.Service.CloseSession()
	case *v1.Command_State:
		reply.Result = &v1.Reply_State{State: snapshot(a.Service.State())}
	case *v1.Command_History:
		limit := int(op.History.GetLimit())
		if limit == 0 {
			limit = 32
		}
		values, cursor, cause := a.Service.History(ctx, op.History.GetBefore(), limit)
		err = cause
		page := &v1.HistoryPage{Before: cursor}
		for _, value := range values {
			page.Items = append(page.Items, historyItem(value))
		}
		reply.Result = &v1.Reply_History{History: page}
	case *v1.Command_Detail:
		page, cause := a.readDetail(ctx, op.Detail)
		err = cause
		reply.Result = &v1.Reply_Detail{Detail: page}
	case *v1.Command_SetModel:
		r := op.SetModel
		value, cause := a.Service.SetModel(ctx, server.ModelSettings{Provider: r.GetProvider(), Endpoint: r.GetEndpoint(), Name: r.GetModel(), Credential: r.GetCredential(), Thinking: r.Thinking, Effort: r.ReasoningEffort, Remember: r.GetRemember()})
		err = cause
		reply.Result = &v1.Reply_State{State: snapshot(value)}
	case *v1.Command_SetSettings:
		r := op.SetSettings
		value, cause := a.Service.Configure(ctx, r.Permission, r.Team)
		err = cause
		reply.Result = &v1.Reply_State{State: snapshot(value)}
	case *v1.Command_Start:
		r := op.Start
		value, cause := a.Service.Start(ctx, r.GetSessionId(), r.GetKey(), r.GetText())
		err = cause
		reply.Result = &v1.Reply_Task{Task: task(value)}
	case *v1.Command_TaskStatus:
		value, cause := a.Service.TaskStatus(ctx, op.TaskStatus.GetTaskId())
		err = cause
		reply.Result = &v1.Reply_Task{Task: task(value)}
	case *v1.Command_Cancel:
		err = a.Service.Cancel(op.Cancel.GetTaskId())
	case *v1.Command_Approve:
		r := op.Approve
		err = a.Service.Approve(r.GetTaskId(), r.GetApprovalId(), r.GetAllow())
	case *v1.Command_Steer:
		r := op.Steer
		err = a.Service.Steer(r.GetTaskId(), r.GetText())
	default:
		err = &server.Error{Code: "invalid_argument", Message: "one supported command is required"}
	}
	if err != nil {
		return a.errorReply(cmd.GetId(), err), nil
	}
	a.sanitize(reply)
	return reply, nil
}

func (a *API) errorReply(id string, err error) *v1.Reply {
	code := "internal"
	var domain *server.Error
	if errors.As(err, &domain) {
		code = domain.Code
	}
	if errors.Is(err, server.ErrTaskNotFound) {
		code = "not_found"
	}
	if errors.Is(err, context.Canceled) {
		code = "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		code = "deadline_exceeded"
	}
	if errors.Is(err, history.ErrReadLimit) || errors.Is(err, admission.ErrCapacity) || errors.Is(err, capacity.ErrLimit) {
		code = "resource_exhausted"
	}
	message := err.Error()
	if a.Redact != nil {
		message = a.Redact(message)
	}
	reply := &v1.Reply{Id: id, Result: &v1.Reply_Error{Error: &v1.Error{Code: code, Message: textPreview(message)}}}
	a.sanitize(reply)
	return reply
}
