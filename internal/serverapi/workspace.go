// Purpose: Dispatch workspace and discovery commands into bounded public read values.
// Pattern: Native wire adapter.
// Usage: API.Call delegates workspace operations here after resource admission.
// Related: api/meca/v1/workspace.proto, internal/commands/server/workspace.go.
package serverapi

import (
	"context"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/processmemory"

	"github.com/keonho-kim/meca/internal/agent/media"
	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func modelSettings(r *v1.ModelRequest) server.ModelSettings {
	return server.ModelSettings{Session: r.GetSessionId(), Provider: r.GetProvider(), Endpoint: r.GetEndpoint(), Name: r.GetModel(), Credential: r.GetCredential(), Thinking: r.Thinking, Effort: r.ReasoningEffort, Remember: r.GetRemember(), DefaultModel: r.GetDefaultModel()}
}
func wireImage(index int, image media.Metadata) *v1.ImageMetadata {
	return &v1.ImageMetadata{Index: int32(index), Path: image.Path, Mime: image.MIME, Width: int32(image.Width), Height: int32(image.Height), Digest: image.Digest, Bytes: image.Bytes}
}

func (a *API) workspaceCall(ctx context.Context, cmd *v1.Command) (*v1.Reply, error) {
	reply := &v1.Reply{Id: cmd.GetId()}
	switch op := cmd.GetOperation().(type) {
	case *v1.Command_Capabilities:
		used, maximum, err := a.Service.StorageUsage(ctx)
		if err != nil {
			return nil, err
		}
		concurrency, language := a.Service.ServerSettings()
		reply.Result = &v1.Reply_Capabilities{Capabilities: &v1.Capabilities{ApiVersion: "meca.v1", Features: []string{"sessions", "tasks", "approvals", "steering", "team-streams", "cursor-replay", "state-pages", "draft-pages", "model-discovery", "workspace-files", "history-images", "skills", "language", "command-activity", "resident-memory", "workspaces", "delete-session", "editor", "file-watch", "browser-commands", "model-diagnostics", "request-lookup", "skill-completion", "tool-execution-identity", "completion-measurements", "editor-probe", "file-mutation-receipts", "workspace-reference-events"}, ImageTypes: []string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/tiff", "image/bmp"}, ImageBytes: media.MaxBytes, InputImageBytes: media.MaxInputBytes, InputImages: media.MaxImages, Concurrency: int64(concurrency), StatePageSize: 100, DetailPageBytes: 64 << 10, Language: language, RequestRecords: used, RequestCapacity: maximum, EditorBytes: fileview.EditorBytes, PromptBytes: server.MaxPromptBytes, CommandBytes: maxRequestBytes}}
	case *v1.Command_Resources:
		value := &v1.Resources{}
		rss, err := processmemory.ResidentBytes()
		if err == nil {
			value.ResidentBytes = &rss
		} else {
			value.ErrorCode = faults.Code(err)
		}
		reply.Result = &v1.Reply_Resources{Resources: value}
	case *v1.Command_Commands:
		values, err := a.Service.Commands(ctx, op.Commands.GetTaskId())
		if err != nil {
			return nil, err
		}
		page := &v1.CommandStates{}
		for _, v := range values {
			var code *int32
			if v.Run.ExitCode != nil {
				value := int32(*v.Run.ExitCode)
				code = &value
			}
			page.Items = append(page.Items, &v1.CommandState{TaskId: v.Task, Agent: v.Agent, Id: v.Run.ID, CallId: v.Run.CallID, Command: textPreview(v.Run.Command), Truncated: len(v.Run.Command) > 16<<10, Status: v.Run.Status, ExitCode: code, ElapsedMillis: v.Run.ElapsedMillis, LastOutputMillis: v.Run.LastOutputMillis})
		}
		reply.Result = &v1.Reply_Commands{Commands: page}
	case *v1.Command_Providers:
		providers := &v1.Providers{}
		for _, p := range connection.Providers() {
			providers.Items = append(providers.Items, &v1.Provider{Id: string(p.Kind), Name: p.Label, Endpoint: p.Endpoint})
		}
		reply.Result = &v1.Reply_Providers{Providers: providers}
	case *v1.Command_DiscoverModels:
		values, err := a.Service.Discover(ctx, server.ModelSettings{Session: op.DiscoverModels.GetSessionId(), Provider: op.DiscoverModels.GetProvider(), Endpoint: op.DiscoverModels.GetEndpoint(), Name: op.DiscoverModels.GetModel(), Credential: op.DiscoverModels.GetCredential()})
		if err != nil {
			return nil, err
		}
		models := &v1.Models{}
		for _, m := range values {
			options := connection.ReasoningOptions(connection.ProviderKind(op.DiscoverModels.GetProvider()), &m)
			models.Items = append(models.Items, &v1.ModelCandidate{Id: m.ID, ContextWindow: int32(m.ContextWindow), ImageInputState: m.ImageInput.State, ImageInputSupported: m.ImageInput.Supported(), ReasoningOn: options.On, ReasoningOff: options.Off, ReasoningLevels: options.Levels, Notice: m.Notice})
		}
		reply.Result = &v1.Reply_Models{Models: models}
	case *v1.Command_Directory:
		r := op.Directory
		limit := int(r.GetLimit())
		if limit == 0 {
			limit = 100
		}
		value, err := a.Service.Directory(ctx, r.GetPath(), int(r.GetOffset()), limit, r.GetRevision())
		if err != nil {
			return nil, err
		}
		ref, _ := a.Service.FileReference(ctx, r.GetPath())
		page := &v1.DirectoryPage{Path: ref, More: value.More, Revision: value.Revision, NextOffset: r.GetOffset() + int32(limit)}
		for _, v := range value.Entries {
			page.Items = append(page.Items, &v1.FileEntry{Name: v.Name, Directory: v.Directory})
		}
		reply.Result = &v1.Reply_Directory{Directory: page}
	case *v1.Command_PreviewFile:
		value, err := a.previewFile(ctx, op.PreviewFile)
		if err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_FilePreview{FilePreview: value}
	case *v1.Command_FileSelections:
		paths, err := a.Service.FileSelections(ctx, op.FileSelections.GetSessionId())
		if err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_FileSelections{FileSelections: &v1.FileSelections{Paths: paths}}
	case *v1.Command_SelectFile:
		r := op.SelectFile
		if err := a.Service.SelectFile(ctx, r.GetSessionId(), r.GetPath(), r.GetSelected()); err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_Ok{Ok: &v1.EmptyResult{}}
	case *v1.Command_HistoryImage:
		r := op.HistoryImage
		items, _, err := a.Service.History(ctx, r.GetSessionId(), r.GetSequence()+1, 1)
		if err != nil {
			return nil, err
		}
		if len(items) != 1 || items[0].Sequence != r.GetSequence() || r.GetIndex() < 0 || int(r.GetIndex()) >= len(items[0].Images) {
			return nil, faults.New(diagcatalog.NotFound, diagcatalog.ServerImageMissing)
		}
		reply.Result = &v1.Reply_Image{Image: wireImage(int(r.GetIndex()), items[0].Images[r.GetIndex()])}
	case *v1.Command_Draft:
		page, err := a.draftPage(ctx, op.Draft)
		if err != nil {
			return nil, err
		}
		reply.Result = &v1.Reply_Draft{Draft: page}
	case *v1.Command_ListTasks:
		r := op.ListTasks
		limit := int(r.GetLimit())
		if limit == 0 {
			limit = 32
		}
		values, next, err := a.Service.Tasks(ctx, r.GetSessionId(), r.GetCursor(), limit)
		if err != nil {
			return nil, err
		}
		page := &v1.TaskPage{NextCursor: next}
		for _, value := range values {
			page.Items = append(page.Items, task(value))
		}
		reply.Result = &v1.Reply_Tasks{Tasks: page}
	default:
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerapiOneSupportedCommandIsRequired)
	}
	return reply, nil
}
