// Purpose: Reuse validated conversation metadata without reopening unchanged databases or WAL files.
// Pattern: Process-bounded immutable metadata index with constant-time LRU maintenance.
// Usage: Readers validate database/WAL identity before using cached summaries; no DB handles are retained.
// Related: internal/storage/sqlite/conversation_reader.go, internal/storage/sqlite/conversation_page.go.
package sqlite

import (
	"container/list"
	"errors"
	"os"
	"sync"
	"time"
)

type metadataEntry struct {
	path, stamp string
	result      conversationResult
	used        time.Time
	bytes       int
}
type conversationMetadata struct {
	mu      sync.Mutex
	entries map[string]*list.Element
	order   list.List
	bytes   int
}

var catalogMetadata conversationMetadata

func conversationStamp(path string, info os.FileInfo) (string, error) {
	stamp := metadataVersion(info)
	wal, err := os.Lstat(path + "-wal")
	if errors.Is(err, os.ErrNotExist) {
		return stamp + "/no-wal", nil
	}
	if err != nil {
		return "", err
	}
	if !wal.Mode().IsRegular() {
		return "", os.ErrInvalid
	}
	// Read-only ATTACH may create an empty WAL; it contains no committed frames.
	if wal.Size() == 0 {
		return stamp + "/no-wal", nil
	}
	return stamp + "/" + metadataVersion(wal), nil
}
func (c *conversationMetadata) remove(element *list.Element) {
	value := element.Value.(*metadataEntry)
	delete(c.entries, value.path)
	c.bytes -= value.bytes
	c.order.Remove(element)
}
func (c *conversationMetadata) get(path, stamp string, now time.Time) (conversationResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	element := c.entries[path]
	if element == nil {
		return conversationResult{}, false
	}
	value := element.Value.(*metadataEntry)
	if value.stamp != stamp || now.Sub(value.used) >= 30*time.Second {
		c.remove(element)
		return conversationResult{}, false
	}
	value.used = now
	c.order.MoveToFront(element)
	return value.result, true
}
func (c *conversationMetadata) put(path, stamp string, result conversationResult, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value := result.conversation
	size := 256 + len(path) + len(stamp) + len(value.ID) + len(value.ProjectRoot) + len(value.DisplayRoot) + len(value.Summary) + len(value.Updated) + len(value.Unavailable)
	if size > 4<<20 {
		return
	}
	if previous := c.entries[path]; previous != nil {
		c.remove(previous)
	}
	for c.order.Len() > 0 {
		last := c.order.Back()
		if len(c.entries) < 4096 && c.bytes+size <= 4<<20 && now.Sub(last.Value.(*metadataEntry).used) < 30*time.Second {
			break
		}
		c.remove(last)
	}
	if c.entries == nil {
		c.entries = make(map[string]*list.Element)
	}
	c.entries[path] = c.order.PushFront(&metadataEntry{path, stamp, result, now, size})
	c.bytes += size
}
