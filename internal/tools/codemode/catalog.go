// Purpose: Derive the model-facing code tool and TypeScript signatures from native definitions.
// Pattern: Pure catalog projection.
// Usage: Agent requests expose this definition while the native catalog validates inner calls.
// Related: internal/agent/tool/schema.go, internal/agent/run/context.go.
package codemode

import (
	"encoding/json"
	"fmt"
	"strings"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

func Definition(tools []toolapi.Definition) toolapi.Definition {
	var description strings.Builder
	description.WriteString(`Execute a JavaScript or TypeScript async function body. Use await tools.<name>({...}) for host operations and return a JSON value. Do not wrap the body in a function or Markdown fences. Declare types inline; import and export are unavailable. TypeScript syntax is transformed, not type-checked. Intermediate tool results remain in the code and execution history; return the facts needed for the next decision. Tool failures reject with an Error whose output contains the tool result. Approval denial, cancellation and persistence failures stop execution and cannot be caught. Code itself needs no approval; individual operations use the existing policy. No filesystem, network, process, module loader or timers are exposed directly. Await every tool call. Independent calls may overlap; each committed result resumes its Promise without waiting for unrelated tools. Promise.all preserves input order; conflicting effects are sequenced by the host. Loading skill instructions or images can end the code early so the model can inspect them before generating more code. Never replay completed effects after an error. Each code call has a fresh context, 64 MiB JS heap, 10 seconds of JS evaluation time, 256 host calls, 32 pending calls and 1 MiB returned JSON. Tool arguments are limited to 4 MiB and individual JSON tool results to 8 MiB.

Available tools (the input contracts below are also checked by the host; results are JSON):
declare const tools: {
`)
	for _, tool := range tools {
		description.WriteString("/** " + strings.ReplaceAll(tool.Description, "*/", "* /") + " */\n")
		description.WriteString(tool.Name + "(args: " + objectType(tool.Parameters) + "): Promise<any>;\n")
	}
	description.WriteString("};")
	return toolapi.Definition{Name: Name, Description: description.String(), Effect: policy.Command,
		Parameters: []toolapi.Parameter{{Name: "code", Description: "JavaScript or TypeScript function body using await tools.<name>(args) and return.", Kind: toolapi.Text, Required: true, MaxBytes: MaxCodeBytes}}}
}

func objectType(parameters []toolapi.Parameter) string {
	var out strings.Builder
	out.WriteString("{\n")
	for _, p := range parameters {
		out.WriteString("/** " + strings.ReplaceAll(p.Description, "*/", "* /") + " */\n")
		switch p.Kind {
		case toolapi.Integer:
			fmt.Fprintf(&out, "/** Integer between 0 and %d. */\n", p.Max)
		case toolapi.TextList, toolapi.ObjectList:
			minimum := 1
			if p.AllowEmpty {
				minimum = 0
			}
			fmt.Fprintf(&out, "/** Between %d and %d items. */\n", minimum, p.Max)
		case toolapi.SingleLineText:
			fmt.Fprintf(&out, "/** At most %d characters on one line. */\n", p.Max)
		}
		if p.MaxBytes > 0 {
			unit := ""
			if p.Kind == toolapi.TextList {
				unit = " per item"
			}
			fmt.Fprintf(&out, "/** UTF-8 byte limit: %d%s. */\n", p.MaxBytes, unit)
		}
		name, _ := json.Marshal(p.Name)
		out.Write(name)
		if !p.Required {
			out.WriteByte('?')
		}
		out.WriteString(": ")
		switch {
		case len(p.Values) != 0:
			for i, value := range p.Values {
				if i > 0 {
					out.WriteString(" | ")
				}
				literal, _ := json.Marshal(value)
				out.Write(literal)
			}
		case p.Kind == toolapi.Integer:
			out.WriteString("number")
		case p.Kind == toolapi.TextList:
			out.WriteString("string[]")
		case p.Kind == toolapi.ObjectList:
			out.WriteString(objectType(p.Fields) + "[]")
		default:
			out.WriteString("string")
		}
		out.WriteString(";\n")
	}
	out.WriteByte('}')
	return out.String()
}
