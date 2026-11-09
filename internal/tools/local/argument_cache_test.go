// Purpose: Keep argument reuse content-bound and isolate mutable preparation slices.
// Pattern: Native input boundary regression tests.
// Usage: go test -race ./internal/tools/local -run TestParsedArguments.
// Related: internal/tools/local/argument_cache.go.
package local

import (
	"encoding/json"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"testing"
	"time"
)

func TestParsedArgumentsRecheckChangedBytes(t *testing.T) {
	e := &Executor{}
	call := toolapi.Call{ID: "call_one", Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"old"}`)}
	_, first, err := e.parsedCall(call)
	if err != nil {
		t.Fatal(err)
	}
	copy(call.Arguments, json.RawMessage(`{"path":"b","content":"new"}`))
	_, second, err := e.parsedCall(call)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != "a" || second.Path != "b" || second.Content != "new" {
		t.Fatal("changed argument body used a stale parse")
	}
	call.ID = ""
	if _, _, err := e.parsedCall(call); err == nil {
		t.Fatal("cache bypassed native identity validation")
	}
}

func TestParsedArgumentsReleasedWhenRequestStops(t *testing.T) {
	e := &Executor{}
	e.BeginCommands(t.Context(), time.Now().Add(time.Minute))
	call := toolapi.Call{ID: "call_request", Name: "write_file", Arguments: json.RawMessage(`{"path":"a","content":"temporary request source"}`)}
	if _, _, err := e.parsedCall(call); err != nil {
		t.Fatal(err)
	}
	if len(e.arguments.values) == 0 {
		t.Fatal("fixture did not exercise retained arguments")
	}
	e.StopCommands()
	if e.arguments.bytes != 0 || len(e.arguments.values) != 0 {
		t.Fatal("finished request retained parsed source")
	}
}
func TestParsedArgumentsOwnPreparationSlices(t *testing.T) {
	e := &Executor{}
	call := toolapi.Call{ID: "call_one", Name: "create_directory", Arguments: json.RawMessage(`{"paths":["a","b"]}`)}
	_, first, err := e.parsedCall(call)
	if err != nil {
		t.Fatal(err)
	}
	first.Paths[0] = "changed"
	_, again, err := e.parsedCall(call)
	if err != nil {
		t.Fatal(err)
	}
	if again.Paths[0] != "a" {
		t.Fatal("preparation mutated the retained argument value")
	}
	e.arguments.clear()
	if e.arguments.bytes != 0 || len(e.arguments.values) != 0 {
		t.Fatal("request retained parsed arguments")
	}
}
