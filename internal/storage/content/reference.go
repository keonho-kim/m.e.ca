// Purpose: Distinguish original JSON values from text views of the same immutable payload.
// Pattern: Explicit stored reference encoding.
// Usage: Pack, restoration and SQLite identity resolution share this single reference contract.
// Related: internal/storage/content/pack.go, internal/storage/content/restore.go, internal/storage/sqlite/prepared_payload.go.
package content

import "strings"

const jsonStringReference = "s:"

func Reference(id string) (physical string, quote bool) {
	return strings.CutPrefix(id, jsonStringReference)
}
func StringReference(id string) string { return jsonStringReference + id }
