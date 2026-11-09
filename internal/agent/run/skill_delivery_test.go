// Purpose: Verify model-visible skill instructions, descriptions and pagination schemas across provider adapters.
// Pattern: Runtime-to-wire contract test with the real local tool catalog.
// Usage: go test ./internal/agent/run -run TestSkillToolDelivery.
// Related: internal/agent/run/start.go, internal/agent/run/skill_pagination_test.go, internal/provider/protocol/request.go.
package run

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol"
)

func skillWireDeclaration(value any, name string) map[string]any {
	switch value := value.(type) {
	case map[string]any:
		if value["name"] == name {
			return value
		}
		for _, key := range []string{"function", "functionDeclarations"} {
			if found := skillWireDeclaration(value[key], name); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range value {
			if found := skillWireDeclaration(child, name); found != nil {
				return found
			}
		}
	}
	return nil
}

func skillWireSchema(declaration map[string]any) map[string]any {
	for _, key := range []string{"parameters", "input_schema", "parametersJsonSchema"} {
		if schema, ok := declaration[key].(map[string]any); ok {
			return schema
		}
	}
	return nil
}

func skillWireInstructions(body map[string]any) string {
	for _, key := range []string{"system", "systemInstruction"} {
		if value := body[key]; value != nil {
			encoded, _ := json.Marshal(value)
			return string(encoded)
		}
	}
	for _, key := range []string{"messages", "input"} {
		items, _ := body[key].([]any)
		for _, item := range items {
			message, _ := item.(map[string]any)
			if message["role"] == "system" {
				encoded, _ := json.Marshal(message["content"])
				return string(encoded)
			}
		}
	}
	return ""
}

func TestSkillToolDelivery(t *testing.T) {
	tools, project := skillDeliveryTools(t)
	g := &generator{replies: []string{"Done"}}
	r := fixture(g, &executor{}, &journal{})
	r.Tools, r.ProjectRoot = tools, project
	if _, err := r.Run(t.Context(), "Inspect this project", approve, emit); err != nil {
		t.Fatal(err)
	}
	if len(g.requests) != 1 {
		t.Fatal("missing model request")
	}
	request := g.requests[0]
	definitions := map[string]toolapi.Definition{}
	for _, definition := range request.Tools {
		definitions[definition.Name] = definition
	}
	names := []string{"list_skills", "load_skill", "read_skill_file"}
	for _, name := range names {
		definition, ok := definitions[name]
		if !ok || definition.Description == "" {
			t.Fatalf("model did not receive %s declaration", name)
		}
		for _, p := range definition.Parameters {
			if p.Description == "" {
				t.Fatalf("%s.%s missing usage description", name, p.Name)
			}
		}
	}
	for _, provider := range connection.Providers() {
		t.Run(string(provider.Kind), func(t *testing.T) {
			var encoder protocol.Encoder
			defer encoder.Clear()
			for range 2 {
				messages, err := encoder.Messages(provider.Kind, request.ModelID, request.Messages)
				if err != nil {
					t.Fatal(err)
				}
				_, body, err := encoder.Encode(provider.Kind, "https://provider.test", nil, request, messages, protocol.Controls{})
				if err != nil {
					t.Fatal(err)
				}
				raw, err := io.ReadAll(body.Reader())
				if err != nil {
					t.Fatal(err)
				}
				var wire map[string]any
				if err := json.Unmarshal(raw, &wire); err != nil {
					t.Fatal(err)
				}
				instructions := skillWireInstructions(wire)
				for _, name := range names {
					if !strings.Contains(instructions, name) {
						t.Fatalf("system instructions omitted %s", name)
					}
					declaration := skillWireDeclaration(wire["tools"], name)
					if declaration == nil || declaration["description"] != definitions[name].Description {
						t.Fatalf("wire omitted %s description", name)
					}
					schema := skillWireSchema(declaration)
					properties, _ := schema["properties"].(map[string]any)
					for _, p := range definitions[name].Parameters {
						property, _ := properties[p.Name].(map[string]any)
						if property["description"] != p.Description {
							t.Fatalf("wire lost %s.%s description", name, p.Name)
						}
					}
					if name == "read_skill_file" {
						offset, _ := properties["offset"].(map[string]any)
						maximum, _ := offset["maximum"].(float64)
						if offset["type"] != "integer" || maximum < 65536 {
							t.Fatalf("schema prevents pagination: %v", offset)
						}
					}
				}
			}
		})
	}
}
