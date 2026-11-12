// Purpose: Own desktop integration prompts and public failure messages.
// Pattern: Central diagnostic catalog.
// Usage: Native folder selection and its authenticated UI boundary reference these messages.
// Related: internal/desktop/folderpicker, internal/serverapi/ui_folder.go.
package catalog

const (
	FolderPickerTitle        Message = "Choose a MECA workspace"
	FolderPickerUnavailable  Message = "native folder selection is unavailable in this environment"
	FolderPickerFailed       Message = "native folder selection failed"
	FolderPickerBusy         Message = "a folder selection dialog is already open"
	FolderPickerLocalOnly    Message = "native folder selection requires a local browser session"
	FolderPickerTimedOut     Message = "native folder selection timed out; use the workspace folder browser"
	FolderPickerFailureCause Message = "native folder selection failed: %w"
)
