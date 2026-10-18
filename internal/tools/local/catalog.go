// Purpose: Declare native tool descriptions, parameter schemas and effects in one catalog.
// Pattern: Simple Module (declarative tool contract).
// Usage: Consumed by native function declarations and local argument validation.
// Related: internal/tools/local/arguments.go, internal/agent/tool/execution.go.
package local

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
)

type specification struct {
	name, label, description string
	fields                   []toolapi.Parameter
	effect                   policy.Effect
}

func filePath() toolapi.Parameter {
	return toolapi.Parameter{Name: "path", Description: "Absolute path or path relative to project_root. Shell cd does not change this base.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}
}
func textField(name, description string, empty bool) toolapi.Parameter {
	return toolapi.Parameter{Name: name, Description: description, Kind: toolapi.Text, Required: true, AllowEmpty: empty, MaxBytes: toolapi.MaxArgumentsBytes}
}
func specifications() []specification {
	specs := []specification{

		{"create_directory", "Creating", "Create one or more directories and missing parents within allowed write paths (mkdir -p). Existing directories succeed without changes. File collisions are rejected. Returns created paths and preserves partial results if interrupted. Use instead of mkdir.", []toolapi.Parameter{{Name: "paths", Description: "Absolute directory paths or paths relative to project_root. One or more paths; shared parents are created once.", Kind: toolapi.TextList, Required: true, Max: 64, MaxBytes: 4096}}, policy.Change},
		{"write_file", "Writing", "Create a UTF-8 file or replace its entire content in an allowed write path; missing parents are created. Inspect an existing file before replacing it. Returns the applied change and file size; a successful write does not verify the resulting program. Use content=empty string for a requested empty file. Use edit_file for partial changes; use this tool instead of touch, shell redirection or heredocs that write source files.", []toolapi.Parameter{filePath(), textField("content", "Complete file content. Preserve whitespace; do not add wrapping code fences.", true)}, policy.Change},
		editSpecification(),
		{"move_file", "Moving", "Move one file or directory within allowed write paths to a nonexistent destination. Existing destinations are rejected. Returns the old and new paths. Use instead of mv.", []toolapi.Parameter{filePath(), {Name: "destination", Description: "Absolute destination or path relative to project_root; shell cd does not change this base.", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}}, policy.Change},
		{"execute_command", "Running", "Run a POSIX shell command for file inspection (cat/head/tail, sed -n, nl), search/listing, builds, tests, package tools (uv init/add/run), and deletion (rm/rmdir/unlink/find -delete/git rm), including binary or recursive deletion. Commands start in project_root with the inherited development environment; cd applies only to this call. Use dedicated create_directory/write_file/edit_file/move_file for their direct operations. The entire command is checked before execution; a rejected command means nothing runs. Wait for prerequisites such as project initialization before reading or modifying their outputs. Returns whether execution started, stdout, stderr, exit code and status; observed file changes cover known targets only. Preserve failing exit codes when combining checks; grep/rg exit 1 means no matches. A running command may apply partial changes before failure or cancellation; inspect those effects before retrying. Permission approval and allowed write paths apply to all child programs.", []toolapi.Parameter{textField("command", "Complete POSIX shell command. Quote literal paths. This call starts in project_root; a previous call's cd has no effect.", false)}, policy.Command},
	}
	for index := range specs {
		if specs[index].name == "edit_file" || specs[index].name == "write_file" {
			specs[index].fields = append(specs[index].fields, toolapi.Parameter{Name: "format_policy", Description: "Default preserve: retain existing EOF/newline format, adding a missing final terminator. literal permits intentional format changes and preserves supplied bytes. Empty/new files use supplied content.", Kind: toolapi.Text, MaxBytes: 8, Values: []string{"preserve", "literal"}})
		}

	}
	return specs
}
func (executor *Executor) Definitions() []toolapi.Definition {
	var out []toolapi.Definition
	for _, spec := range specifications() {
		if spec.name == "execute_command" {
			spec.description += " Writable roots for all tools: " + executor.redact(strings.Join(executor.boundary.Writes(), ", ")) + "."
		}
		out = append(out, toolapi.Definition{Name: spec.name, Description: spec.description, Parameters: spec.fields, Effect: spec.effect})
	}
	return out
}
