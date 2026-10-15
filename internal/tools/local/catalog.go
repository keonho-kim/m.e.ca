// Purpose: Declare native tool descriptions, parameter schemas and effects in one catalog.
// Pattern: Simple Module (declarative tool contract).
// Usage: Consumed by native function declarations and local argument validation.
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
	return toolapi.Parameter{Name: "path", Description: "File or directory path relative to the project root; use . for the root.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
}
func textField(name, description string, empty bool) toolapi.Parameter {
	return toolapi.Parameter{Name: name, Description: description, Kind: toolapi.Text, Required: true, AllowEmpty: empty, MaxBytes: toolapi.MaxArgumentsBytes}
}
func numberField(name, description string, max int) toolapi.Parameter {
	return toolapi.Parameter{Name: name, Description: description + " Omit or use zero for the default.", Kind: toolapi.Integer, Max: max}
}
func specifications() []specification {
	page := []toolapi.Parameter{filePath(), numberField("offset", "Starting byte offset.", 100000000), numberField("limit", "Maximum bytes to read.", 16*1024*1024)}
	specs := []specification{
		{"read_file", "Reading", "Read source lines with line#hash anchors. Finish this call before composing dependent edit_file arguments; a first read and its dependent edit cannot share a batch.", page, policy.Read},
		{"create_directory", "Creating", "Use for directory creation instead of mkdir: creates all paths and missing parents (mkdir -p), accepting existing directories.", []toolapi.Parameter{{Name: "paths", Description: "Directory paths relative to the project root. One or more paths; shared parents are created once.", Kind: toolapi.TextList, Required: true, Max: 64, MaxBytes: 4096}}, policy.Change},
		{"write_file", "Writing", "Create a new UTF-8 file or replace an existing file with the supplied complete content. Use content=empty string for a requested empty file; missing parents are created. Use edit_file for anchored partial changes.", []toolapi.Parameter{filePath(), textField("content", "Complete file content. Preserve whitespace; do not add wrapping code fences.", true)}, policy.Change},
		{"edit_file", "Editing", "Replace a range using anchors from a completed read_file call. Copy only source into new_text. Default preserve retains the line boundary and newline style.", []toolapi.Parameter{filePath(), {Name: "start_anchor", Description: "Copy the first line#8hex anchor from read_file. Use the literal anchor string.", Kind: toolapi.LineAnchor, Required: true, MaxBytes: 32}, {Name: "end_anchor", Description: "Copy the last line#8hex anchor from the same read. For one line, repeat start_anchor.", Kind: toolapi.LineAnchor, Required: true, MaxBytes: 32}, textField("new_text", "Exact replacement lines. Exclude anchor labels; preserve whitespace and line endings. Empty means delete the selected lines.", true)}, policy.Change},
		{"delete_file", "Deleting", "Delete a file or empty directory", []toolapi.Parameter{filePath()}, policy.Change},
		{"move_file", "Moving", "Move without overwriting the destination", []toolapi.Parameter{filePath(), {Name: "destination", Description: "Destination path relative to the project root.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}}, policy.Change},
		{"execute_command", "Running", "Run search/listing commands (find, ls, rg, grep), builds, tests or package tools. Use read_file for source reads; mkdir uses create_directory, touch/truncate uses write_file, rm uses delete_file, mv uses move_file, and content edits use edit_file. Do not resubmit rejected direct file operations as shell wrappers. Shell syntax, pipelines and redirects follow the host shell.", []toolapi.Parameter{textField("command", "Exact shell command to run.", false)}, policy.Command},
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
