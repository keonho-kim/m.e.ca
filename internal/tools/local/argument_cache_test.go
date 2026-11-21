// Purpose: Keep argument reuse content-bound and isolate mutable preparation slices.
// Pattern: Native input boundary regression tests.
// Usage: go test -race ./internal/tools/local -run TestParsedArguments.
// Related: internal/tools/local/argument_cache.go.
package local

import (
	"encoding/json"
	"testing"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
)

func TestArgumentAdmissionSharesOnlyMatchingCatalogValidation(t *testing.T) {
	e := &Executor{}
	definition := toolcatalog.Definitions("write_file")[0]
	call := toolapi.Call{ID: "call_one", Name: "write_file", Arguments: json.RawMessage(`{"reason":"Update local fixture","path":"a","content":"old"}`)}
	if err := e.ValidateCallArguments(call, definition); err != nil {
		t.Fatal(err)
	}
	if len(e.arguments.values) != 1 {
		t.Fatal("admission did not populate the preparation cache")
	}
	_, parsed, err := e.parsedCall(call)
	if err != nil || parsed.Reason != "Update local fixture" || len(e.arguments.values) != 1 {
		t.Fatalf("preparation did not reuse admission: %+v %v", parsed, err)
	}
	definition.Parameters[0].Max = 5
	if err := e.ValidateCallArguments(call, definition); err == nil {
		t.Fatal("cached local arguments bypassed the advertised reason limit")
	}
}

func TestParsedArgumentsRecheckChangedBytes(t *testing.T) {
	e := &Executor{}
	call := toolapi.Call{ID: "call_one", Name: "write_file", Arguments: json.RawMessage(`{"reason":"Replace fixture data","path":"a","content":"old"}`)}
	_, first, err := e.parsedCall(call)
	if err != nil {
		t.Fatal(err)
	}
	copy(call.Arguments, json.RawMessage(`{"reason":"Replace fixture data","path":"b","content":"new"}`))
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
	call := toolapi.Call{ID: "call_request", Name: "write_file", Arguments: json.RawMessage(`{"reason":"Create temporary fixture","path":"a","content":"temporary request source"}`)}
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
