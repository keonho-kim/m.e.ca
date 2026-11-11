// Purpose: Own interactive workspace, browser authentication and editor failures.
// Pattern: Central diagnostic catalog.
// Usage: WebUI HTTP adapters, workspace ownership and file editing use these messages.
// Related: internal/serverapi/ui_auth.go, internal/commands/server/workspaces.go, internal/fileview/editor.go.
package catalog

const (
	UIFolderNameInvalid     Message = "provide one non-empty folder name up to 255 UTF-8 bytes without path separators"
	UIFolderExists          Message = "a file or folder already exists with that name"
	UIFolderChanged         Message = "the parent folder changed; browse it again before creating the folder"
	UIFileMutationInvalid   Message = "provide a file or folder, its directory revision and one valid change"
	UIFileMutationChanged   Message = "the folder changed; refresh it before changing its entries"
	UIFileMutationExists    Message = "an entry with that name already exists in the destination folder"
	UIFlagHelp              Message = "serve the embedded browser workspace"
	UIWorkspaceInvalid      Message = "select an existing accessible workspace directory"
	UIWorkspaceUnknown      Message = "workspace is not registered"
	UIWorkspaceBusy         Message = "workspace resources are currently in use"
	UIWorkspaceClosed       Message = "workspace server is shutting down"
	UIEditorReadOnly        Message = "this file cannot be edited: too large, truncated, masked, binary or unsupported text"
	UIEditorChanged         Message = "the file changed; reload its current revision before saving"
	UIEditorInvalid         Message = "provide a complete UTF-8 document and its original revision"
	UICommandUnavailable    Message = "this command is not available in the browser"
	UIAuthRejected          Message = "browser authentication failed; enter the server token again"
	UIAuthCapacity          Message = "browser session capacity is occupied"
	UIBuildMissing          Message = "embedded UI assets are missing; run the frontend build before building MECA"
	UIDeleteActive          Message = "stop the active task before deleting this conversation"
	UIDeleteBusy            Message = "conversation readers are still active; retry after they finish"
	UIDeletePending         Message = "conversation deletion is incomplete; retry deletion before starting work"
	UIDiagnosticUnavailable Message = "model diagnostics are unavailable"
)
