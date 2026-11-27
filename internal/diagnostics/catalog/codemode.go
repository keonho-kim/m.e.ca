// Purpose: Own code execution, bridge and resource failure messages.
// Pattern: Central diagnostic catalog.
// Usage: The code runtime and agent execution owner report these failures.
// Related: internal/tools/codemode/engine.go, internal/agent/run/codemode.go.
package catalog

const (
	CodeInvalidInput     Message = "code must be nonempty UTF-8 and no larger than %d bytes"
	CodeTransform        Message = "TypeScript conversion failed: %s"
	CodeRuntime          Message = "code execution failed: %s"
	CodeResourceLimit    Message = "code execution exceeded its %s limit"
	CodeInvalidBridge    Message = "invalid code tool bridge request"
	CodeUnknownTool      Message = "code requested unavailable tool %q"
	CodeUnawaited        Message = "code returned with unfinished tool calls; await every tool call before returning"
	CodeUnresolved       Message = "code is waiting on a Promise with no pending tool calls; timers and external I/O are unavailable"
	CodeInvalidResult    Message = "code must return a JSON-serializable value: %s"
	CodeDispatchMismatch Message = "code tool dispatcher returned an unknown or missing result"
	CodeModelBoundary    Message = "new model input is available; generate the next code using the returned instructions or attachments"
	CodeOnlyTool         Message = "call codemode with JavaScript or TypeScript code; use tools.<name>(arguments) inside the code"
)
