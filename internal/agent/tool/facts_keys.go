// Purpose: Identify stable known file observations for request progress.
// Pattern: Pure semantic projection.
// Usage: Request progress observations compare the same stable fact values.
// Related: internal/agent/tool/facts.go, internal/agent/run/semantic_progress.go.
package tool

import (
	"encoding/json"
	"sort"
)

func FileKeys(files []FileFact) []string {
	keys := map[string]bool{}
	for _, file := range files {
		file.Changed = false
		file.SourceID = ""
		file.ObservedRun = ""
		file.Revision = 0
		body, _ := json.Marshal(file)
		keys["file:"+string(body)] = true
	}
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
