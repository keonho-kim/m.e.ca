// Purpose: Declare native tool descriptions, parameter schemas and effects in one catalog.
// Pattern: Simple Module (declarative tool contract).
// Usage: Consumed by native function declarations and local argument validation.
// Related: internal/tools/local/arguments.go, internal/agent/tool/execution.go.
package local

import (
	"slices"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
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

// The private catalog is immutable after package initialization.
var catalog = buildSpecifications()

func buildSpecifications() []specification {
	specs := []specification{

		{"create_directory", "Creating", "Create one or more directories and missing parents (mkdir -p). Existing directories succeed without changes. File collisions are rejected. Returns created paths and preserves partial results if interrupted.", []toolapi.Parameter{{Name: "paths", Description: "Absolute directory paths or paths relative to project_root. One or more paths; shared parents are created once.", Kind: toolapi.TextList, Required: true, Max: 64, MaxBytes: 4096}}, policy.Change},
		{"write_file", "Writing", "Create a UTF-8 file or replace its entire content; missing parents are created. Inspect an existing file before replacing it. Empty content creates or empties the file. Returns the applied change and file size. Use edit_file for partial changes.", []toolapi.Parameter{filePath(), textField("content", "Complete file content. Preserve whitespace; do not add wrapping code fences.", true)}, policy.Change},
		editSpecification(),
		{"execute_command", "Running", "Run a POSIX shell command in project_root with the inherited host environment. Supports &&, ||, pipelines, command substitutions and multiple statements; cd applies only to this call. Use for file inspection, search, builds, tests, package management and host file operations, including mv for moving or renaming files and directories. Deleting or moving project_root itself or its ancestors, protected root/system mutations and privilege escalation are blocked. Prefer file tools for precise edits and diff previews. Use && when later steps depend on earlier success, or cd path || exit before subsequent statements. Use the supplied project_root or an observed path for directory changes. A POSIX pipeline reports its last command's exit code. Preserve failures; grep/rg exit 1 means no matches. Returns status, exit code and stdout/stderr. Large output includes excerpts and session log paths readable with cat/sed. Failures or cancellation can leave partial effects; inspect current state before retrying.", []toolapi.Parameter{textField("command", "Complete POSIX shell command. Quote literal paths. This call starts in project_root; a previous call's cd has no effect.", false)}, policy.Command},
	}
	for index := range specs {
		if specs[index].name == "edit_file" || specs[index].name == "write_file" {
			specs[index].fields = append(specs[index].fields, toolapi.Parameter{Name: "format_policy", Description: "Default preserve: retain existing EOF/newline format, adding a missing final terminator. literal permits intentional format changes and preserves supplied bytes. Empty/new files use supplied content.", Kind: toolapi.Text, MaxBytes: 8, Values: []string{"preserve", "literal"}})
		}

	}
	return specs
}
func (*Executor) Definitions() []toolapi.Definition {
	out := make([]toolapi.Definition, len(catalog))
	for i, spec := range catalog {
		out[i] = toolapi.Definition{Name: spec.name, Description: spec.description, Parameters: cloneParameters(spec.fields), Effect: spec.effect}
	}
	return out
}

// Definitions crosses a mutable-slice boundary; private validation schemas must
// remain independent of modifications by generators and protocol adapters.
func cloneParameters(parameters []toolapi.Parameter) []toolapi.Parameter {
	copy := slices.Clone(parameters)
	for i := range copy {
		copy[i].Fields = cloneParameters(copy[i].Fields)
		copy[i].Values = slices.Clone(copy[i].Values)
	}
	return copy
}
