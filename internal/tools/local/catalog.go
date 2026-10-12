// Purpose: Declare the fixed tools and their ordered literal questions in one catalog.
// Pattern: Simple Module (declarative tool contract).
// Usage: Consumed by staged model dialogue and local argument validation.
// Related: internal/tools/local/arguments.go, internal/agent/tools.go.
package local

import (
	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/policy"
)

type specification struct {
	name, label, description string
	fields                   []agent.ToolParameter
	effect                   policy.Effect
}

func filePath() agent.ToolParameter {
	return agent.ToolParameter{Name: "path", Question: "Which file or directory? Reply only with its literal path, without quotes.", Kind: agent.LiteralPath, Required: true, MaxBytes: 4096}
}
func textField(name, question string, empty bool) agent.ToolParameter {
	return agent.ToolParameter{Name: name, Question: question, Kind: agent.Text, Required: true, AllowEmpty: empty, MaxBytes: agent.MaxArgumentsBytes}
}
func numberField(name, question string, max int) agent.ToolParameter {
	return agent.ToolParameter{Name: name, Question: question + " Reply with one integer, or empty for the default zero.", Kind: agent.Integer, Max: max}
}
func specifications() []specification {
	page := []agent.ToolParameter{filePath(), numberField("offset", "Starting byte or directory entry offset?", 100000000), numberField("limit", "Maximum bytes or directory entries?", 256*1024)}
	return []specification{
		{"read_file", "Reading", "Read a bounded UTF-8 file range", page, policy.Read},
		{"list_directory", "Listing", "List a directory page", []agent.ToolParameter{filePath(), numberField("offset", "Starting entry offset?", 25000), numberField("limit", "Maximum entries?", 1000)}, policy.Read},
		{"search_files", "Searching", "Find filenames matching a glob", []agent.ToolParameter{filePath(), textField("pattern", "Which filename glob? Reply only with the glob.", false)}, policy.Read},
		{"grep", "Searching", "Search file lines using a regular expression", []agent.ToolParameter{filePath(), textField("pattern", "Which regular expression? Reply only with the expression.", false)}, policy.Read},
		{"write_file", "Writing", "Write exact UTF-8 content", []agent.ToolParameter{filePath(), textField("content", "What exact content should the file contain? Preserve whitespace; do not add wrapping code fences.", true)}, policy.Change},
		{"edit_file", "Editing", "Replace exactly one occurrence", []agent.ToolParameter{filePath(), textField("old_text", "Which exact existing text should be replaced? Preserve whitespace.", false), textField("new_text", "What exact replacement text? Preserve whitespace; empty means remove the old text.", true)}, policy.Change},
		{"delete_file", "Deleting", "Delete a file or empty directory", []agent.ToolParameter{filePath()}, policy.Change},
		{"move_file", "Moving", "Move without overwriting the destination", []agent.ToolParameter{filePath(), {Name: "destination", Question: "Which exact destination path, without quotes?", Kind: agent.LiteralPath, Required: true, MaxBytes: 4096}}, policy.Change},
		{"execute_command", "Running", "Run a shell command with policy and cancellation", []agent.ToolParameter{textField("command", "Which exact shell command? Reply only with the command, not an explanation.", false), numberField("timeout_seconds", "Timeout in seconds?", 600)}, policy.Command},
	}
}
func (*Executor) Definitions() []agent.ToolDefinition {
	var out []agent.ToolDefinition
	for _, spec := range specifications() {
		out = append(out, agent.ToolDefinition{Name: spec.name, Description: spec.description, Parameters: spec.fields, Effect: spec.effect})
	}
	return out
}
