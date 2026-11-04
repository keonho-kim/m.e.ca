// Purpose: Verify public discussion and shared-contract native argument boundaries.
// Pattern: Public tool contract tests.
// Usage: go test ./internal/tools/team/catalog -run TestDiscussion.
// Related: internal/tools/team/catalog/catalog.go, internal/tools/team/catalog/arguments.go.
package catalog

import (
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestDiscussionAndContractArguments(t *testing.T) {
	for _, call := range []toolapi.Call{
		{Name: "discuss", Arguments: []byte(`{"message":"The existing API uses 423.","reply_to":184,"mentions":["Hana"]}`)},
		{Name: "discuss", Arguments: []byte(`{"message":"I found an alternative implementation."}`)},
		{Name: "set_work", Arguments: []byte(`{"summary":"Implement login errors","paths":["src/auth"],"contract":"Locked accounts return 423","reply_to":185}`)},
	} {
		if _, err := Parse(call); err != nil {
			t.Fatalf("%s: %v", call.Name, err)
		}
	}
	for _, arguments := range []string{
		`{"message":"Check this","reply_to":-1}`,
		`{"message":"Check this","reply_to":"184"}`,
		`{"message":"Check this","sender":"Hana"}`,
		`{"message":"Check this","mentions":["Hana",3]}`,
	} {
		if _, err := Parse(toolapi.Call{Name: "discuss", Arguments: []byte(arguments)}); err == nil {
			t.Fatal("invalid discussion accepted:", arguments)
		}
	}
}
