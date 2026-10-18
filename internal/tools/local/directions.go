// Purpose: Give actionable native-tool routing feedback for rejected direct shell operations.
// Pattern: Pure tool-catalog guidance.
// Usage: Prepare uses the selected catalog tool to explain the next valid call.
// Related: internal/tools/local/catalog.go, internal/policy/shell/check.go.
package local

func dedicatedAdvice(name string) string {
	return "Nothing from the rejected shell command ran. Submit the allowed shell work separately, then use a separate file-tool call. " + dedicatedCall(name)
}

func dedicatedCall(name string) string {
	switch name {
	case "create_directory":
		return "Call create_directory with paths: an array of directory paths. It creates missing parents and accepts existing directories."
	case "write_file":
		return "Call write_file with path and complete content; an empty string creates an empty file. Read an existing file before replacing it."
	case "edit_file":
		return "Inspect the source with execute_command (cat or sed -n), then call edit_file with path and edits: an array of {old_text,new_text} replacements."
	case "move_file":
		return "Call native move_file with path and destination. It never overwrites an existing destination."
	default:
		return "Use the named dedicated tool as a native function. Complete dependent file operations before submitting a later shell command."
	}
}
