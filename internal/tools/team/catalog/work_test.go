// Purpose: Keep work release, exact teammate names and native schemas aligned.
// Pattern: Public argument contract tests.
// Usage: Catalog tests cover empty scope release without weakening directory creation.
// Related: internal/tools/team/catalog/catalog.go, internal/agent/tool/json_schema.go.
package catalog

import (
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	localcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
)

func TestWorkReleaseAndNamedRestorationArguments(t *testing.T) {
	if _, err := Parse(toolapi.Call{Name: "set_work", Arguments: []byte(`{"summary":"Handing files to Hana","paths":[]}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(toolapi.Call{Name: "add_team_member", Arguments: []byte(`{"message":"Review the API","member_name":"Hana"}`)}); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(toolapi.Call{Name: "add_team_member", Arguments: []byte(`{"message":"Review","agent_id":"Hana"}`)}); err == nil {
		t.Fatal("obsolete argument accepted")
	}
	definitions := localcatalog.Definitions("create_directory")
	if _, err := toolapi.ValidateArguments(definitions[0], []byte(`{"paths":[]}`)); err == nil {
		t.Fatal("work release weakened directory arguments")
	}
	definition, _ := definition("set_work")
	schema := definition.JSONSchema()["properties"].(map[string]any)["paths"].(map[string]any)
	if schema["minItems"] != 0 {
		t.Fatal("provider schema rejects scope release")
	}
}
