// Purpose: Convert a bounded TypeScript function body into executable JavaScript.
// Pattern: Compiler adapter without module resolution or filesystem access.
// Usage: Run compiles the full code before creating any host tool calls.
// Related: internal/tools/codemode/engine.go, internal/tools/codemode/contract.go.
package codemode

import (
	"strings"
	"unicode/utf8"

	"github.com/evanw/esbuild/pkg/api"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func transform(code string) (string, error) {
	if len(code) > MaxCodeBytes || strings.TrimSpace(code) == "" || !utf8.ValidString(code) {
		return "", faults.New(diag.InvalidArgument, diag.CodeInvalidInput, MaxCodeBytes)
	}
	result := api.Transform("(async function(tools) {\n"+code+"\n})", api.TransformOptions{
		Loader: api.LoaderTS, Target: api.ES2022, Sourcefile: "codemode.ts",
		LogLevel: api.LogLevelSilent, Charset: api.CharsetUTF8,
	})
	if len(result.Errors) != 0 {
		problem := result.Errors[0]
		if problem.Location != nil && problem.Location.Line > 1 {
			problem.Location.Line--
		}
		messages := api.FormatMessages([]api.Message{problem}, api.FormatMessagesOptions{Kind: api.ErrorMessage, Color: false})
		return "", faults.New(diag.InvalidArgument, diag.CodeTransform, strings.Join(messages, "\n"))
	}
	return string(result.Code), nil
}
