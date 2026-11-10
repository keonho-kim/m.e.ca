// Purpose: Share bounded immutable thumbnail bytes while coalescing concurrent derivations.
// Pattern: Transport-owned derived-image cache with cancellable single-flight reads.
// Usage: historyImage revalidates the source and holds its session lease before calling get.
// Related: internal/serverapi/history_image.go, internal/media/thumbnail/render.go.
package serverapi

import (
	"context"
	"sync"
	"time"
)

type thumbnailEntry struct {
	data []byte
	used time.Time
}
type thumbnailCache struct {
	mu      sync.Mutex
	entries map[string]thumbnailEntry
	pending map[string]chan struct{}
	bytes   int
}

func (c *thumbnailCache) get(ctx context.Context, key string, load func() ([]byte, error)) ([]byte, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		c.mu.Lock()
		now := time.Now()
		for key, value := range c.entries {
			if now.Sub(value.used) >= 5*time.Minute {
				delete(c.entries, key)
				c.bytes -= len(value.data)
			}
		}
		if value, ok := c.entries[key]; ok {
			value.used = now
			c.entries[key] = value
			c.mu.Unlock()
			return value.data, nil
		}
		if done := c.pending[key]; done != nil {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-done:
				continue
			}
		}
		if c.pending == nil {
			c.pending = make(map[string]chan struct{})
		}
		done := make(chan struct{})
		c.pending[key] = done
		c.mu.Unlock()
		body, err := load()
		if err == nil {
			err = ctx.Err()
		}
		c.mu.Lock()
		delete(c.pending, key)
		close(done)
		if err == nil && len(body) <= 4<<20 {
			for len(c.entries) >= 32 || c.bytes+len(body) > 4<<20 {
				var oldest string
				for key, value := range c.entries {
					if oldest == "" || value.used.Before(c.entries[oldest].used) {
						oldest = key
					}
				}
				c.bytes -= len(c.entries[oldest].data)
				delete(c.entries, oldest)
			}
			if c.entries == nil {
				c.entries = make(map[string]thumbnailEntry)
			}
			c.entries[key] = thumbnailEntry{body, time.Now()}
			c.bytes += len(body)
		}
		c.mu.Unlock()
		return body, err
	}
}
