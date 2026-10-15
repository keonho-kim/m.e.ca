// Purpose: Give actionable native-tool routing feedback for rejected direct shell operations.
// Pattern: Pure tool-catalog guidance.
// Usage: Prepare uses the selected catalog tool to explain the next valid call.
// Related: internal/tools/local/catalog.go, internal/policy/shell/check.go.
package local

func dedicatedAdvice(name string) string {
	switch name {
	case "create_directory":
		return "Call native create_directory with paths: an array of directory paths. It creates missing parents and accepts existing directories. Then create files with write_file. Nothing from the rejected shell command ran."
	case "write_file":
		return "Call native write_file with path and complete content. Use an empty content string for a requested empty file. This replaces file content, so read existing files before changing them. Nothing from the rejected shell command ran."
	case "read_file":
		return "Call native read_file with path. Its result includes line anchors for a later edit_file call. Use execute_command only for listing/search and other command work."
	case "edit_file":
		return "Read the source using read_file first; in a later native edit_file call copy its start_anchor/end_anchor and provide plain source in new_text. Do not copy display labels."
	case "move_file":
		return "Call native move_file with path and destination. It never overwrites an existing destination."
	case "delete_file":
		return "Call native delete_file with path. Do not resubmit the deletion as a shell command."
	default:
		return "Use the named dedicated tool as a native function. Complete dependent file operations before submitting a later shell command."
	}
}
