// Purpose: Declare native tool descriptions, parameter schemas and effects in one catalog.
// Pattern: Simple Module (declarative tool contract).
// Usage: Consumed by native function declarations and local argument validation.
// Related: internal/tools/local/arguments.go, internal/agent/tool/execution.go.
package local

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/policy/shell"
)

type specification struct {
	name, label, description string
	fields                   []toolapi.Parameter
	effect                   policy.Effect
}

func filePath() toolapi.Parameter {
	return toolapi.Parameter{Name: "path", Description: "File or directory path relative to the project root; use . for the root.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
}
func textField(name, description string, empty bool) toolapi.Parameter {
	return toolapi.Parameter{Name: name, Description: description, Kind: toolapi.Text, Required: true, AllowEmpty: empty, MaxBytes: toolapi.MaxArgumentsBytes}
}
func numberField(name, description string, max int) toolapi.Parameter {
	return toolapi.Parameter{Name: name, Description: description + " Omit or use zero for the default.", Kind: toolapi.Integer, Max: max}
}
func specifications() []specification {
	page := []toolapi.Parameter{filePath(), numberField("offset", "Starting byte offset.", 100000000), numberField("limit", "Maximum bytes to read.", 256*1024)}
	specs := []specification{
		{"read_file", "Reading", "Read numbered UTF-8 lines with short edit anchors", page, policy.Read},
		{"create_directory", "Creating", "Create one empty directory; parent must exist", []toolapi.Parameter{filePath()}, policy.Change},
		{"write_file", "Writing", "Create a new UTF-8 file or replace an existing file with the supplied complete content. Parent directory must exist. Use edit_file for anchored partial changes.", []toolapi.Parameter{filePath(), textField("content", "Complete file content. Preserve whitespace; do not add wrapping code fences.", true)}, policy.Change},
		{"edit_file", "Editing", "Replace a previously read line range using its start/end anchors", []toolapi.Parameter{filePath(), {Name: "start_anchor", Description: "Copy the first line#8hex anchor from read_file. Use the literal anchor string.", Kind: toolapi.LineAnchor, Required: true, MaxBytes: 32}, {Name: "end_anchor", Description: "Copy the last line#8hex anchor from the same read. For one line, repeat start_anchor.", Kind: toolapi.LineAnchor, Required: true, MaxBytes: 32}, textField("new_text", "Exact replacement lines. Exclude anchor labels; preserve whitespace and line endings. Empty means delete the selected lines.", true)}, policy.Change},
		{"delete_file", "Deleting", "Delete a file or empty directory", []toolapi.Parameter{filePath()}, policy.Change},
		{"move_file", "Moving", "Move without overwriting the destination", []toolapi.Parameter{filePath(), {Name: "destination", Description: "Destination path relative to the project root.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}}, policy.Change},
		{"execute_command", "Running", "Run search/listing commands (find, ls, rg, grep), builds, tests or package tools with policy and cancellation. Use read_file before anchored edits and dedicated tools for file changes. Plan queries use built-in reads, basename globs and no gitignore filtering; they never execute a shell. " + shell.PlanQueryHelp, []toolapi.Parameter{textField("command", "Exact shell command to run.", false)}, policy.Command},
	}
	for index := range specs {
		if specs[index].name == "edit_file" || specs[index].name == "write_file" {
			specs[index].fields = append(specs[index].fields, toolapi.Parameter{Name: "format_policy", Description: "Default preserve: retain existing EOF/newline format, adding a missing final terminator. literal permits intentional format changes and preserves supplied bytes. Empty/new files use supplied content.", Kind: toolapi.Text, MaxBytes: 8, Values: []string{"preserve", "literal"}})
		}

	}
	return specs
}
func (*Executor) Definitions() []toolapi.Definition {
	var out []toolapi.Definition
	for _, spec := range specifications() {
		out = append(out, toolapi.Definition{Name: spec.name, Description: spec.description, Parameters: spec.fields, Effect: spec.effect})
	}
	return out
}
