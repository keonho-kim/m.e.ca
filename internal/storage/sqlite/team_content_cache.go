// Purpose: Reuse canonical public-text identities while private readers keep the room in use.
// Pattern: Bounded owner-scoped immutable metadata cache.
// Usage: Private context commits share encoding work; the final child Close releases retained text.
// Related: internal/storage/sqlite/team_payloads.go, internal/storage/sqlite/team_sessions.go.
package sqlite

import (
	"encoding/json"
	"slices"
	"sync"

	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
)

const teamContentBytes = 2 * 1024 * 1024
const teamContentEntries = 128

type teamTextEncoding struct {
	text, id string
	size     int
}
type teamContentCache struct {
	mu           sync.Mutex
	users, bytes int
	values       map[int64]teamTextEncoding
	order        []int64
}

func encodeTeamText(text string) (teamTextEncoding, error) {
	raw, err := json.Marshal(text)
	if err != nil {
		return teamTextEncoding{}, err
	}
	return teamTextEncoding{text: text, id: recordcontent.ID(raw), size: len(raw)}, nil
}

func (c *teamContentCache) encoding(sequence int64, text string) (teamTextEncoding, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if value, ok := c.values[sequence]; ok && value.text == text {
		return value, nil
	}
	value, err := encodeTeamText(text)
	if err != nil || c.users == 0 || len(text) > teamContentBytes {
		return value, err
	}
	if previous, ok := c.values[sequence]; ok {
		c.bytes -= len(previous.text)
		delete(c.values, sequence)
		index := slices.Index(c.order, sequence)
		c.order = slices.Delete(c.order, index, index+1)
	}
	for len(c.order) > 0 && (len(c.order) >= teamContentEntries || c.bytes+len(text) > teamContentBytes) {
		old := c.order[0]
		c.bytes -= len(c.values[old].text)
		delete(c.values, old)
		c.order = slices.Delete(c.order, 0, 1)
	}
	if c.values == nil {
		c.values = make(map[int64]teamTextEncoding)
	}
	c.values[sequence] = value
	c.order = append(c.order, sequence)
	c.bytes += len(text)
	return value, nil
}

func (c *teamContentCache) acquire() {
	c.mu.Lock()
	c.users++
	c.mu.Unlock()
}
func (c *teamContentCache) release() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users--
	if c.users == 0 {
		c.values, c.order, c.bytes = nil, nil, 0
	}
}
func (c *teamContentCache) clear() {
	c.mu.Lock()
	c.values, c.order, c.bytes = nil, nil, 0
	c.mu.Unlock()
}
