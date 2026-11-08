// Purpose: Reject ambiguous restoration paths and preserve exact numbers across shared array patches.
// Pattern: Pure content transformation contract tests.
// Usage: go test ./internal/storage/content -run TestRestoreTree.
// Related: internal/storage/content/restore.go, internal/storage/content/pack.go.
package content

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRestoreTreeRejectsOverlappingAndNoncanonicalPaths(t *testing.T) {
	for _, refs := range []string{
		`{"[\"items\"]":"a","[\"items\",\"0\"]":"b"}`,
		`{"[\"items\",\"01\"]":"a"}`,
		`{"[\"items\",\"+1\"]":"a"}`,
		`{"[\"items\",\"3\"]":"a"}`,
	} {
		if _, err := Restore([]byte(`{"items":[null,null]}`), []byte(refs), nil, func(string) (json.RawMessage, error) { return json.RawMessage(`null`), nil }); err == nil {
			t.Fatal("ambiguous path accepted", refs)
		}
	}
}

func TestRestoreTreeLoadsSharedPayloadOnceAndPreservesIntegers(t *testing.T) {
	reads := 0
	body, err := Restore([]byte(`{"items":[null,null],"number":9007199254740993}`), []byte(`{"[\"items\",\"0\"]":"same","[\"items\",\"1\"]":"same"}`), nil, func(string) (json.RawMessage, error) {
		reads++
		return json.RawMessage(`{"id":18446744073709551615}`), nil
	})
	if err != nil || reads != 1 || !strings.Contains(string(body), "9007199254740993") || strings.Count(string(body), "18446744073709551615") != 2 {
		t.Fatal(string(body), reads, err)
	}
}
