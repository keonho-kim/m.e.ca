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
	fmt.Fprintf(&description, `Run a program when its execution contributes computation or host operations needed for the user's requested outcome. Deliberation and composing a conversational response happen outside this execution tool. Return computed or observed results, rather than routing an already-written response through code.

Supply a JavaScript or TypeScript async function body, without a function wrapper or Markdown fences. The tools object is already available; call its functions with the declared object arguments. These functions exist only inside codemode, not as separate model-facing tools. Declare TypeScript types inline; import and export are unavailable. TypeScript is transformed, not type-checked.

The normal model-facing result is the program's explicit return value. For a single operation, use return await tools.<name>({...}); for composed work, capture the results and return the facts needed for your next decision, including evidence for claims of success. Other host outputs remain in execution history.

A completed code call reports return_status=value for a defined JSON value, including explicit null. If it ends with undefined, return_status=undefined identifies the missing value and fallback.outputs supplies successful committed tool outputs in completion order, within a combined %d KiB JSON budget. Outputs that fit are preserved completely; oversized values become {excerpt,truncated:true} using the longest UTF-8 prefix that fits their encoded JSON budget. fallback.truncated means output was shortened or omitted. These are observed tool outputs, not the program's computed return value. Their operations already ran; do not rerun them just to recover a missing return. Return explicitly to select what enters model context.

Await every tool call. Independent calls may overlap; each committed result resumes its Promise without waiting for unrelated tools. Promise.all preserves input order and the host sequences conflicting effects. A skill instruction or image can end this code with status needs_model. Read that supplied input before writing a new code call for dependent work. Operations already completed at that boundary are not replayed.

Tool failures reject with an Error whose output contains the tool result. Approval denial, cancellation and persistence failures stop execution and cannot be caught. Inspect completed and partial results before deciding whether a further operation is needed. Code itself needs no approval; each host operation uses its existing policy.

No filesystem, network, process, module loader or timers are exposed directly. Each code call has a fresh context, 64 MiB JS heap, 10 seconds of JS evaluation time, 256 host calls, 32 pending calls and 1 MiB returned JSON. Tool arguments are limited to 4 MiB and individual JSON tool results to 8 MiB.

Available host functions (the input contracts below are checked by the host; results are JSON):
declare const tools: {
`, MaxFallbackBytes>>10)
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
