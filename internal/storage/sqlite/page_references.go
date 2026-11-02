// Purpose: Identify the last record referencing each payload within one bounded page.
// Pattern: Pure page-local reference index.
// Usage: History hydration and export use this index without reading payload bodies.
// Related: internal/storage/sqlite/page_payloads.go, internal/storage/sqlite/record_codec.go.
package sqlite

import "encoding/json"

// Index growth is independent of accepted record sizes and reference counts.
// Unindexed payloads still load normally, without temporary reuse.
const pageReferenceEntries = 256

func indexPayloadRefs(last map[string]int, index int, refsBody []byte) (map[string]int, error) {
	if string(refsBody) == "{}" {
		return last, nil
	}
	var refs map[string]string
	if err := json.Unmarshal(refsBody, &refs); err != nil {
		return nil, err
	}
	for _, id := range refs {
		if _, tracked := last[id]; !tracked && len(last) >= pageReferenceEntries {
			continue
		}
		if last == nil {
			last = make(map[string]int)
		}
		last[id] = index
	}
	return last, nil
}
