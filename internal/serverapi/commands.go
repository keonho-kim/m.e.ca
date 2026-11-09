// Purpose: Dispatch typed external commands to the server use cases without owning execution state.
// Pattern: Transport boundary adapter.
// Usage: HTTP, WebSocket and Protobuf RPC use the same API.Call implementation.
// Related: api/meca/v1/control.proto, internal/commands/server/service.go.
package serverapi

import (
	"context"
	"sync"
	"time"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	"github.com/keonho-kim/meca/internal/serverapi/detail"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/proto"
)

type API struct {
	Service          *server.Service
	Redact           func(string) string
	NewMask          func() func(string, bool) string
	RedactPreview    func(string) string
	RedactRange      func(string, int, int) (string, error)
	RedactionVersion func() int
	Details          *detail.Cache
	readOnce         sync.Once
	reads            *admission.Budget
	browseOnce       sync.Once
	browse           *admission.Budget
}

func (a *API) Call(ctx context.Context, cmd *v1.Command) (result *v1.Reply, cause error) {
	if err := admission.ResizeInput(ctx, int64(proto.Size(cmd))+4096); err != nil {
		return a.errorReply(cmd.GetId(), err), nil
	}
	started := time.Now()
	ctx = diagnostics.WithRedactor(ctx, a.Redact)
	operation := "unknown"
	if cmd != nil {
		message := cmd.ProtoReflect()
		if field := message.WhichOneof(message.Descriptor().Oneofs().ByName("operation")); field != nil {
			operation = string(field.Name())
		}
	}
	defer func() {
		code := result.GetError().GetCode()
		diagnostics.Log(ctx, diagcatalog.ControlCompleted, "component", "serverapi", "operation", operation, "duration_ms", time.Since(started).Milliseconds(), "error_code", code, "task_id", result.GetTask().GetId())
	}()
	reply := &v1.Reply{Id: cmd.GetId(), Result: &v1.Reply_Ok{Ok: &v1.EmptyResult{}}}
	if len(cmd.GetId()) > 128 {
		return a.errorReply("", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerapiCommandIDExceedsBytes)), nil
	}
	var err error
	switch cmd.GetOperation().(type) {
	case *v1.Command_History, *v1.Command_Detail, *v1.Command_HistoryImage, *v1.Command_PreviewFile:
		release, ok := a.readLimit().Try(1)
		if !ok {
			return a.errorReply(cmd.GetId(), faults.New(diagcatalog.ResourceExhausted, diagcatalog.ServerOriginalReadBusy)), nil
		}
		defer release()
	case *v1.Command_ListSessions, *v1.Command_ListTasks, *v1.Command_ListSkills, *v1.Command_GetSkill, *v1.Command_ReadSkillFile, *v1.Command_Directory, *v1.Command_DiscoverModels:
		release, ok := a.browseLimit().Try(1)
		if !ok {
			return a.errorReply(cmd.GetId(), faults.New(diagcatalog.ResourceExhausted, diagcatalog.ServerBrowseBusy)), nil
		}
		defer release()
	}
	switch op := cmd.GetOperation().(type) {
	case *v1.Command_ListSkills:
		err = a.listSkills(ctx, op.ListSkills, reply)
	case *v1.Command_GetSkill:
		err = a.getSkill(ctx, op.GetSkill, reply)
	case *v1.Command_ReadSkillFile:
		err = a.readSkillFile(ctx, op.ReadSkillFile, reply)
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
		err = a.Service.CloseSession(ctx, op.CloseSession.GetSessionId())
	case *v1.Command_State:
		value, cause := a.Service.StatePage(ctx, op.State.GetSessionId(), op.State.GetTaskId(), op.State.GetAfterTask(), int(op.State.GetLimit()))
		err = cause
		reply.Result = &v1.Reply_State{State: snapshot(value)}
	case *v1.Command_History:
		limit := int(op.History.GetLimit())
		if limit == 0 {
			limit = 32
		}
		values, cursor, cause := a.Service.History(ctx, op.History.GetSessionId(), op.History.GetBefore(), limit)
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
		value, cause := a.Service.SetModel(ctx, modelSettings(r))
		err = cause
		reply.Result = &v1.Reply_State{State: snapshot(value)}
	case *v1.Command_SetSettings:
		r := op.SetSettings
		var value server.Snapshot
		var cause error
		if r.Language != nil && r.Permission == nil && r.Team == nil {
			value, cause = a.Service.SetLanguage(ctx, r.GetLanguage())
		} else if r.Language != nil {
			cause = faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerChangeExactlyOneSettingPerCommand)
		} else {
			value, cause = a.Service.Configure(ctx, r.GetSessionId(), r.Permission, r.Team)
		}
		err = cause
		reply.Result = &v1.Reply_State{State: snapshot(value)}
	case *v1.Command_Start:
		r := op.Start
		value, cause := a.Service.StartAttachments(ctx, r.GetSessionId(), r.GetKey(), r.GetText(), r.GetAttachmentIds(), r.GetPaths())
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
		err = a.Service.SteerPaths(ctx, r.GetTaskId(), r.GetText(), r.GetPaths())
	default:
		reply, err = a.workspaceCall(ctx, cmd)
	}
	if err != nil {
		return a.errorReply(cmd.GetId(), err), nil
	}
	a.sanitize(reply)
	return reply, nil
}

func (a *API) errorReply(id string, err error) *v1.Reply {
	code := diagcatalog.NativeCode(faults.Code(err))
	message := faults.PublicMessage(err, a.Redact)
	reply := &v1.Reply{Id: id, Result: &v1.Reply_Error{Error: &v1.Error{Code: code, Message: textPreview(message)}}}
	a.sanitize(reply)
	return reply
}
