// Purpose: Declare the fixed tools and their ordered literal questions in one catalog.
// Pattern: Simple Module (declarative tool contract).
// Usage: Consumed by staged model dialogue and local argument validation.
// Related: internal/tools/local/arguments.go, internal/agent/tool/execution.go.
package local

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

type specification struct {
	name, label, description string
	fields                   []toolapi.Parameter
	effect                   policy.Effect
}

func filePath() toolapi.Parameter {
	return toolapi.Parameter{Name: "path", Question: "Which file or directory? Reply only with its literal path, without quotes.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
}
func textField(name, question string, empty bool) toolapi.Parameter {
	return toolapi.Parameter{Name: name, Question: question, Kind: toolapi.Text, Required: true, AllowEmpty: empty, MaxBytes: toolapi.MaxArgumentsBytes}
}
func numberField(name, question string, max int) toolapi.Parameter {
	return toolapi.Parameter{Name: name, Question: question + " Reply with one integer, or empty for the default zero.", Kind: toolapi.Integer, Max: max}
}
func specifications() []specification {
	page := []toolapi.Parameter{filePath(), numberField("offset", "Starting byte or directory entry offset?", 100000000), numberField("limit", "Maximum bytes or directory entries?", 256*1024)}
	return []specification{
		{"read_file", "Reading", "Read numbered UTF-8 lines with short edit anchors", page, policy.Read},
		{"list_directory", "Listing", "List a directory page", []toolapi.Parameter{filePath(), numberField("offset", "Starting entry offset?", 25000), numberField("limit", "Maximum entries?", 1000)}, policy.Read},
		{"search_files", "Searching", "Find filenames matching a glob", []toolapi.Parameter{filePath(), textField("pattern", "Which filename glob? Reply only with the glob.", false)}, policy.Read},
		{"grep", "Searching", "Search file lines using a regular expression", []toolapi.Parameter{filePath(), textField("pattern", "Which regular expression? Reply only with the expression.", false)}, policy.Read},
		{"create_directory", "Creating", "Create one empty directory; parent must exist", []toolapi.Parameter{filePath()}, policy.Change},
		{"write_file", "Writing", "Write exact UTF-8 content", []toolapi.Parameter{filePath(), textField("content", "What exact content should the file contain? Preserve whitespace; do not add wrapping code fences.", true)}, policy.Change},
		{"edit_file", "Editing", "Replace a previously read line range using its start/end anchors", []toolapi.Parameter{filePath(), {Name: "start_anchor", Question: "Copy the first line#8hex anchor from read_file. Return only that anchor.", Kind: toolapi.LineAnchor, Required: true, MaxBytes: 32}, {Name: "end_anchor", Question: "Copy the last line#8hex anchor from the same read. For one line, repeat start_anchor.", Kind: toolapi.LineAnchor, Required: true, MaxBytes: 32}, textField("new_text", "What exact replacement lines? Exclude anchor labels; preserve whitespace and line endings. Empty means delete the selected lines.", true)}, policy.Change},
		{"delete_file", "Deleting", "Delete a file or empty directory", []toolapi.Parameter{filePath()}, policy.Change},
		{"move_file", "Moving", "Move without overwriting the destination", []toolapi.Parameter{filePath(), {Name: "destination", Question: "Which exact destination path, without quotes?", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}}, policy.Change},
		{"execute_command", "Running", "Run a shell command with policy and cancellation", []toolapi.Parameter{textField("command", "Which exact shell command? Reply only with the command, not an explanation.", false)}, policy.Command},
	}
}
func (*Executor) Definitions() []toolapi.Definition {
	var out []toolapi.Definition
	for _, spec := range specifications() {
		out = append(out, toolapi.Definition{Name: spec.name, Description: spec.description, Parameters: spec.fields, Effect: spec.effect})
	}
	return out
}
