// Purpose: Reuse immutable directory ordering under explicit byte, entry and revision bounds.
// Pattern: Reader-owned metadata cache with filesystem revalidation on every page.
// Usage: Reader.Directory copies one page and rechecks each visible target separately.
// Related: internal/fileview/directory.go, internal/fileview/reader.go.
package fileview

import (
	"context"
	"crypto/sha256"
	"fmt"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"os"
	"sync"
	"time"
)

type DirectoryPage struct {
	Entries  []Entry
	More     bool
	Revision string
}
type directorySnapshot struct {
	revision string
	entries  []Entry
	bytes    int
	used     time.Time
}
type directorySnapshots struct {
	mu      sync.Mutex
	values  map[string]directorySnapshot
	pending map[string]chan struct{}
	bytes   int
}

func directoryRevision(info os.FileInfo) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s/%d/%d/%d", directoryIdentity(info), info.Size(), info.ModTime().UnixNano(), info.Mode())))
	return fmt.Sprintf("%x", sum)
}
func (c *directorySnapshots) load(ctx context.Context, target Target, expected string) ([]Entry, string, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		info, err := target.Root.Stat(target.Relative)
		if err != nil {
			return nil, "", err
		}
		if !info.IsDir() {
			return nil, "", fault.Errorf(diag.FileviewInvalidDirectoryPage)
		}
		revision := directoryRevision(info)
		if expected != "" && expected != revision {
			return nil, "", fault.New(diag.Conflict, diag.ServerReadRevision)
		}
		c.mu.Lock()
		now := time.Now()
		for key, value := range c.values {
			if now.Sub(value.used) >= 30*time.Second {
				delete(c.values, key)
				c.bytes -= value.bytes
			}
		}
		if value, ok := c.values[target.Absolute]; ok && value.revision == revision {
			value.used = now
			c.values[target.Absolute] = value
			c.mu.Unlock()
			return value.entries, revision, nil
		}
		if done := c.pending[target.Absolute]; done != nil {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-done:
				continue
			}
		}
		if c.pending == nil {
			c.pending = make(map[string]chan struct{})
		}
		done := make(chan struct{})
		c.pending[target.Absolute] = done
		c.mu.Unlock()
		entries, err := readDirectoryEntries(ctx, target)
		if err == nil {
			var after os.FileInfo
			after, err = target.Root.Stat(target.Relative)
			if err == nil && directoryRevision(after) != revision {
				err = fault.New(diag.Conflict, diag.ServerReadRevision)
			}
		}
		c.mu.Lock()
		delete(c.pending, target.Absolute)
		close(done)
		if err == nil {
			c.retain(target.Absolute, revision, entries, time.Now())
		}
		c.mu.Unlock()
		return entries, revision, err
	}
}
func (c *directorySnapshots) retain(path, revision string, entries []Entry, now time.Time) {
	bytes := len(path) + cap(entries)*32
	for _, entry := range entries {
		bytes += len(entry.Name)
	}
	if old, ok := c.values[path]; ok {
		delete(c.values, path)
		c.bytes -= old.bytes
	}
	if bytes > 4<<20 {
		return
	}
	for len(c.values) >= 8 || c.bytes+bytes > 4<<20 {
		var oldest string
		for key, value := range c.values {
			if oldest == "" || value.used.Before(c.values[oldest].used) {
				oldest = key
			}
		}
		c.bytes -= c.values[oldest].bytes
		delete(c.values, oldest)
	}
	if c.values == nil {
		c.values = make(map[string]directorySnapshot)
	}
	c.values[path] = directorySnapshot{revision, entries, bytes, now}
	c.bytes += bytes
}
