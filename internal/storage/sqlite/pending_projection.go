// Purpose: Keep only unfinished call objects and their payload references before hydration.
// Pattern: Pure stored-record projection preserving original call array positions.
// Usage: PendingTools avoids loading completed arguments from a partially finished native batch.
// Related: internal/storage/sqlite/pending_tools.go, internal/storage/content/restore.go.
package sqlite

import (
	"fmt"
	"strconv"

	json "github.com/goccy/go-json"
)

func (group pendingToolGroup) selectedRecord() (storedRecord, error) {
	var message struct {
		Calls []json.RawMessage `json:"calls"`
	}
	if err := json.Unmarshal(group.record.body, &message); err != nil {
		return storedRecord{}, err
	}
	selected := make([]bool, len(message.Calls))
	for _, position := range group.indices {
		if position < 0 || position >= len(selected) {
			return storedRecord{}, fmt.Errorf("stored pending call position is invalid")
		}
		selected[position] = true
	}
	for position := range message.Calls {
		if !selected[position] {
			message.Calls[position] = nil
		}
	}
	var refs map[string]string
	if err := json.Unmarshal(group.record.refs, &refs); err != nil {
		return storedRecord{}, err
	}
	for key := range refs {
		var path []string
		if err := json.Unmarshal([]byte(key), &path); err != nil || len(path) == 0 {
			return storedRecord{}, fmt.Errorf("invalid stored payload path")
		}
		if path[0] != "calls" {
			delete(refs, key)
			continue
		}
		if len(path) < 3 {
			return storedRecord{}, fmt.Errorf("invalid stored call payload path")
		}
		position, err := strconv.Atoi(path[1])
		if err != nil || position < 0 || position >= len(selected) {
			return storedRecord{}, fmt.Errorf("invalid stored call payload position")
		}
		if !selected[position] {
			delete(refs, key)
		}
	}
	record := group.record
	var err error
	record.body, err = json.Marshal(message)
	if err != nil {
		return storedRecord{}, err
	}
	record.refs, err = json.Marshal(refs)
	return record, err
}
