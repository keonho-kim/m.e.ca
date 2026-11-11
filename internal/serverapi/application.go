// Purpose: Dispatch workspace management and explicit interactive operations.
// Pattern: Native wire adapter over application-owned use cases.
// Usage: API.Call delegates extensions before the existing workspace commands.
// Related: api/meca/v1/application.proto, internal/commands/server/workspaces.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/commands/server"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"path/filepath"
)

func (a *API) scoped(service *server.Service, id string) *API {
	return &API{Workspaces: a.Workspaces, ScopeID: id, Remember: a.Remember, Service: service, Redact: a.Redact, NewMask: a.NewMask, RedactPreview: a.RedactPreview, RedactRange: a.RedactRange, RedactionVersion: a.RedactionVersion, Details: a.Details, reads: a.readLimit(), browse: a.browseLimit()}
}

func (a *API) applicationCall(ctx context.Context, cmd *v1.Command) (*v1.Reply, error) {
	reply := &v1.Reply{Id: cmd.GetId(), Result: &v1.Reply_Ok{Ok: &v1.EmptyResult{}}}
	switch op := cmd.GetOperation().(type) {
	case *v1.Command_ListWorkspaces:
		if a.Workspaces == nil {
			return nil, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
		}
		values, defaultID := a.Workspaces.List()
		page := &v1.Workspaces{DefaultId: defaultID}
		for _, v := range values {
			page.Items = append(page.Items, &v1.Workspace{Id: v.ID, Path: v.Path, Name: v.Name, ActiveTasks: int32(v.Active), PendingApprovals: int32(v.Approvals), Error: v.Error})
		}
		reply.Result = &v1.Reply_Workspaces{Workspaces: page}
	case *v1.Command_AddWorkspace:
		if a.Workspaces == nil {
			return nil, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
		}
		if _, err := a.Workspaces.Add(ctx, op.AddWorkspace.GetPath()); err != nil {
			return nil, err
		}
		return a.applicationCall(ctx, &v1.Command{Id: cmd.GetId(), Operation: &v1.Command_ListWorkspaces{ListWorkspaces: &v1.Empty{}}})
	case *v1.Command_BrowseFolders:
		if a.Workspaces == nil {
			return nil, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
		}
		path, err := a.Workspaces.ResolvePath(op.BrowseFolders.GetPath())
		if err != nil {
			return nil, err
		}
		value, err := a.Service.Directory(ctx, path, int(op.BrowseFolders.GetOffset()), 200, "")
		if err != nil {
			return nil, err
		}
		page := &v1.DirectoryPage{Path: path, Parent: filepath.Dir(path), More: value.More, Revision: value.Revision, NextOffset: op.BrowseFolders.GetOffset() + 200}
		for _, entry := range value.Entries {
			if entry.Directory {
				page.Items = append(page.Items, &v1.FileEntry{Name: entry.Name, Directory: true})
			}
		}
		reply.Result = &v1.Reply_Directory{Directory: page}
	case *v1.Command_CreateFolder:
		if a.Workspaces == nil {
			return nil, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
		}
		parent, err := a.Workspaces.ResolvePath(op.CreateFolder.GetPath())
		if err != nil {
			return nil, err
		}
		path, err := a.Service.CreateFolder(ctx, parent, op.CreateFolder.GetName())
		if err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_Directory{Directory: &v1.DirectoryPage{Path: path, Parent: filepath.Dir(path)}}
	case *v1.Command_MutateFile:
		request := op.MutateFile
		value, err := a.Service.MutateEntry(ctx, request.GetSessionId(), fileview.EntryMutation{
			Path: request.GetPath(), Revision: request.GetRevision(), Name: request.GetName(),
			Folder: request.GetFolder(), Remove: request.GetRemove(),
		})
		if err != nil {
			return nil, err
		}
		result := &v1.FileMutationResult{Path: value.Path, Paths: value.Paths, ReferencesReady: value.ReferencesReady}
		if value.Failure != nil {
			result.ErrorCode = diag.NativeCode(fault.Code(value.Failure))
			result.ErrorMessage = fault.PublicMessage(value.Failure, a.Redact)
		}
		reply.Result = &v1.Reply_FileMutation{FileMutation: result}
	case *v1.Command_SessionUsage:
		total, err := a.Service.SessionUsage(ctx, op.SessionUsage.GetSessionId())
		if err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_Usage{Usage: &v1.Usage{Input: total.Input, Output: total.Output, Cost: total.Cost}}
	case *v1.Command_DeleteSession:
		if err := a.Service.DeleteSession(ctx, op.DeleteSession.GetSessionId()); err != nil {
			return nil, err
		}
	case *v1.Command_CommandCatalog:
		page := &v1.CommandCatalog{}
		for _, v := range commands.BrowserCatalog() {
			page.Items = append(page.Items, &v1.CommandDescription{Name: v.Name, Description: v.Description})
		}
		reply.Result = &v1.Reply_CommandCatalog{CommandCatalog: page}
	case *v1.Command_ParseInput:
		action, argument, err := commands.BrowserInput(op.ParseInput.GetText())
		if err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_ParsedInput{ParsedInput: &v1.ParsedInput{Action: action, Argument: argument}}
	case *v1.Command_EditorRead, *v1.Command_EditorSave:
		r, save := cmd.GetEditorRead(), false
		if op, ok := op.(*v1.Command_EditorSave); ok {
			r, save = op.EditorSave, true
		}
		value, err := a.Service.Editor(ctx, r.GetPath(), r.GetText(), r.GetRevision(), save)
		if err != nil {
			return nil, err
		}
		if save {
			value.Text = ""
		}
		reply.Result = &v1.Reply_Editor{Editor: &v1.EditorDocument{Path: value.Path, Text: value.Text, Revision: value.Revision, Editable: value.Editable, Reason: value.Reason}}
	case *v1.Command_LookupRequest:
		value, err := a.Service.LookupRequest(ctx, op.LookupRequest.GetSessionId(), op.LookupRequest.GetKey())
		if err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_Task{Task: task(value)}
	case *v1.Command_SuggestSkills:
		values, err := a.Service.SuggestSkills(ctx, op.SuggestSkills.GetPrefix())
		if err != nil {
			return nil, err
		}
		page := &v1.Skills{}
		for _, value := range values {
			page.Items = append(page.Items, wireSkill(value))
		}
		reply.Result = &v1.Reply_Skills{Skills: page}
	case *v1.Command_DiagnoseModel:
		r := op.DiagnoseModel
		value, err := a.Service.Diagnose(ctx, server.ModelSettings{Session: r.GetSessionId(), Provider: r.GetProvider(), Endpoint: r.GetEndpoint(), Name: r.GetModel(), Credential: r.GetCredential()})
		if err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_Diagnostic{Diagnostic: &v1.DiagnosticResult{Chat: value.Chat, Reasoning: value.Reasoning, Thinking: value.Thinking}}
	default:
		return a.workspaceCall(ctx, cmd)
	}
	return reply, nil
}
