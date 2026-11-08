// Purpose: Identify the last record referencing each payload within one bounded page.
// Pattern: Pure page-local reference index.
// Usage: History hydration and export use this index without reading payload bodies.
// Related: internal/storage/sqlite/page_payloads.go, internal/storage/sqlite/record_codec.go.
package sqlite

import (
	"encoding/json"

	"github.com/keonho-kim/meca/internal/storage/content"
)

// Index growth is independent of accepted record sizes and reference counts.
// Unindexed payloads still load normally, without temporary reuse.
const pageReferenceEntries = 256

func indexPayloadRefs(last map[payloadKey]int, store *Store, index int, refsBody []byte) (map[payloadKey]int, error) {
	if string(refsBody) == "{}" {
		return last, nil
	}
	var refs map[string]string
	if err := json.Unmarshal(refsBody, &refs); err != nil {
		return nil, err
	}
	for _, id := range refs {
		id, _ = content.Reference(id)
		key := payloadKey{store, id}
		if _, tracked := last[key]; !tracked && len(last) >= pageReferenceEntries {
			continue
		}
		if last == nil {
			last = make(map[payloadKey]int)
		}
		last[key] = index
	}
	return last, nil
}
