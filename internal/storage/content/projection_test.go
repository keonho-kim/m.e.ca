// Purpose: Verify nested projection omits opaque payload reads and preserves selected native display fields.
// Pattern: Selective-hydration boundary regression.
// Usage: Run TestNestedProjection.
// Related: internal/storage/content/projection.go, internal/storage/content/restore.go.
package content

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNestedProjectionLoadsOnlyVisiblePayloads(t *testing.T) {
	source, _ := json.Marshal(map[string]any{"kind": "lat_message", "message": map[string]any{
		"role": "assistant", "content": strings.Repeat("visible ", 8000), "reasoning_state": strings.Repeat("opaque ", 8000),
		"calls": []any{map[string]any{"id": "call_1", "name": "execute_command", "arguments": map[string]any{"command": strings.Repeat("hidden ", 8000)}}},
	}})
	values := map[string]json.RawMessage{}
	packed, refs, err := Pack(source, func(raw []byte) (string, error) {
		id := ID(raw)
		values[id] = append([]byte(nil), raw...)
		return id, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	loads := 0
	body, err := Restore(packed, refs, map[string]bool{"kind": true, "message": true}, func(id string) (json.RawMessage, error) { loads++; return values[id], nil }, Selection{"message": {
		"role": nil, "content": nil, "calls": {"*": {"id": nil, "name": nil}},
	}})
	if err != nil || loads != 1 {
		t.Fatalf("selected projection loaded %d payloads: %v", loads, err)
	}
	if !strings.Contains(string(body), "visible") || strings.Contains(string(body), "opaque") || strings.Contains(string(body), "hidden") {
		t.Fatal("nested field projection changed visibility")
	}
}
