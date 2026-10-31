// Purpose: Keep allocation-free fact sizing equivalent to the persisted JSON representation.
// Pattern: Differential domain contract test.
// Usage: go test ./internal/agent/tool -run TestFactsSize.
// Related: internal/agent/tool/facts_size.go.
package tool

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFactsSize(t *testing.T) {
	for _, path := range []string{"file.go", "한글/파일.go", "<path>\n&\t", string([]byte{0xff}), strings.Repeat("a", 4096)} {
		for _, full := range []bool{false, true} {
			f := Facts{Files: []FileFact{{Path: path, Version: "v1", Changed: full, Stale: !full}}}
			if full {
				f.Files[0].Observation = "present"
				f.Files[0].ObservedRun = "run"
				f.Files[0].SourceID = "source"
				f.Files[0].Revision = 1234
				f.Files = append(f.Files, FileFact{Path: "second", Version: "v2"})
			}
			body, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			if got := factsBytes(f); got != len(body) {
				t.Fatalf("size=%d want=%d %s", got, len(body), body)
			}
		}
	}
}
