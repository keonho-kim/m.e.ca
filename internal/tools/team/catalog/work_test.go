// Purpose: Keep work release, exact teammate names and native schemas aligned.
// Pattern: Public argument contract tests.
// Usage: Catalog tests cover empty scope release without weakening directory creation.
// Related: internal/tools/team/catalog/catalog.go, internal/agent/tool/json_schema.go.
package catalog

import (
	"encoding/json"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	localcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
)

func TestWorkReleaseAndNamedRestorationArguments(t *testing.T) {
	if _, err := Parse(toolapi.Call{Name: "set_work", Arguments: []byte(`{"summary":"Handing files to Hana","paths":[]}`)}); err != nil {
		t.Fatal(err)
	}
	invitation, err := Parse(toolapi.Call{Name: "add_team_member", Arguments: []byte(`{"domain":"api-review","message":"Review the API","member_name":"Hana"}`)})
	if err != nil || invitation.Domain != "api-review" {
		t.Fatal(err)
	}
	if _, err := Parse(toolapi.Call{Name: "add_team_member", Arguments: []byte(`{"message":"Review","member_name":"Hana"}`)}); err == nil {
		t.Fatal("missing invitation domain accepted")
	}
	definitions := localcatalog.Definitions("create_directory")
	if _, err := toolapi.ValidateArguments(definitions[0], []byte(`{"paths":[]}`)); err == nil {
		t.Fatal("work release weakened directory arguments")
	}
	definition, _ := definition("set_work")
	body, err := json.Marshal(definition.JSONSchema())
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]struct{ MinItems *int }
	}
	if err := json.Unmarshal(body, &schema); err != nil {
		t.Fatal(err)
	}
	if minItems := schema.Properties["paths"].MinItems; minItems == nil || *minItems != 0 {
		t.Fatal("provider schema rejects scope release")
	}
}

func TestWorkDomainArgumentsShareProposalReferences(t *testing.T) {
	invite, err := Parse(toolapi.Call{Name: "add_team_member", Arguments: []byte(`{"domain":"api","work_ref":9,"message":"Review compatibility"}`)})
	if err != nil || invite.WorkRef != 9 {
		t.Fatalf("invite did not accept a public work reference: %+v %v", invite, err)
	}
	for _, body := range []string{
		`{"summary":"Review compatibility","paths":[],"domains":[{"domain":"api","work_ref":9},{"domain":"storage"}]}`,
		`{"summary":"Release work","paths":[],"domains":[]}`,
	} {
		declared, err := Parse(toolapi.Call{Name: "set_work", Arguments: []byte(body)})
		if err != nil || declared.Domains == nil {
			t.Fatalf("work declaration lost explicit domains: %+v %v", declared, err)
		}
	}
	if _, err := Parse(toolapi.Call{Name: "set_work", Arguments: []byte(`{"summary":"Review","paths":[],"domains":[{"work_ref":9}]}`)}); err == nil {
		t.Fatal("work domain without a shared key was accepted")
	}
	for _, exported := range Definitions() {
		if exported.Name != "set_work" {
			continue
		}
		for i, parameter := range exported.Parameters {
			if parameter.Name == "domains" {
				exported.Parameters[i].Fields[0].Name = "changed by consumer"
				original, _ := definition("set_work")
				if original.Parameters[i].Fields[0].Name != "domain" {
					t.Fatal("nested domain schema lost ownership")
				}
				return
			}
		}
	}
	t.Fatal("set_work did not advertise domains")
}
