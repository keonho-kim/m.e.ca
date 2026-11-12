// Purpose: Share immutable masked event projections and JSON across simultaneous observers.
// Pattern: Bounded observer-lifetime cache with per-event computation ownership.
// Usage: Transports reuse cursor projections without holding the registry lock during encoding.
// Related: internal/serverapi/watch.go, internal/serverapi/http.go, internal/serverapi/projection.go.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sync"
)

const projectionBytes = 1 << 20

type projectedEvent struct {
	project, encode sync.Once
	value           *v1.Notification
	json            []byte
	err             error
	order           uint64
	bytes           int
	jsonCharged     bool
}
type eventProjectionCache struct {
	mu                        sync.Mutex
	observers, version, bytes int
	order                     uint64
	values                    map[string]*projectedEvent
}

func (c *eventProjectionCache) observe() func() {
	c.mu.Lock()
	c.observers++
	c.mu.Unlock()
	return sync.OnceFunc(func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.observers--
		if c.observers < 2 {
			c.values = nil
			c.bytes = 0
		}
	})
}

func (a *API) projectEvent(value server.Event) *v1.Notification {
	c := &a.events
	c.mu.Lock()
	version := a.redactionVersion()
	if c.version != version {
		c.values = nil
		c.bytes = 0
		c.version = version
	}
	cache := c.observers > 1 && a.RedactionVersion != nil && value.Cursor != ""
	var entry *projectedEvent
	if cache {
		entry = c.values[value.Cursor]
		if entry == nil {
			for len(c.values) >= 128 || c.bytes+1024 > projectionBytes {
				c.evictLocked("")
			}
			if c.values == nil {
				c.values = make(map[string]*projectedEvent)
			}
			c.order++
			entry = &projectedEvent{order: c.order, bytes: 1024}
			c.values[value.Cursor] = entry
			c.bytes += entry.bytes
		}
	}
	c.mu.Unlock()
	build := func() *v1.Notification {
		notification := &v1.Notification{Body: &v1.Notification_Event{Event: event(value)}}
		a.sanitize(notification)
		return notification
	}
	if !cache {
		return build()
	}
	entry.project.Do(func() {
		notification := build()
		size := proto.Size(notification) + 1024
		c.mu.Lock()
		entry.value = notification
		if c.values[value.Cursor] == entry && c.version == version {
			if size > projectionBytes || a.redactionVersion() != version {
				c.bytes -= entry.bytes
				delete(c.values, value.Cursor)
			} else {
				for c.bytes+size-entry.bytes > projectionBytes {
					c.evictLocked(value.Cursor)
				}
				c.bytes += size - entry.bytes
				entry.bytes = size
			}
		}
		c.mu.Unlock()
	})
	return entry.value
}

// evictLocked drops only retained references; current encoders still own their entry.
func (c *eventProjectionCache) evictLocked(keep string) {
	var oldest string
	for key, item := range c.values {
		if key != keep && (oldest == "" || item.order < c.values[oldest].order) {
			oldest = key
		}
	}
	if oldest != "" {
		c.bytes -= c.values[oldest].bytes
		delete(c.values, oldest)
	}
}

func (a *API) redactionVersion() int {
	if a.RedactionVersion == nil {
		return 0
	}
	return a.RedactionVersion()
}

func (a *API) notificationJSON(value *v1.Notification) ([]byte, error) {
	c := &a.events
	cursor := notificationCursor(value)
	c.mu.Lock()
	version := a.redactionVersion()
	entry := c.values[cursor]
	if c.version != version || entry != nil && entry.value != value {
		entry = nil
	}
	c.mu.Unlock()
	if entry == nil {
		// A detached projection may predate a credential revision even after a
		// different observer has already refreshed the registry's version.
		if a.Redact != nil {
			value = proto.Clone(value).(*v1.Notification)
			a.sanitize(value)
		}
		return protojson.Marshal(value)
	}
	entry.encode.Do(func() { entry.json, entry.err = protojson.Marshal(value) })
	c.mu.Lock()
	current := version == a.redactionVersion()
	if current && c.values[cursor] == entry && entry.err == nil && !entry.jsonCharged {
		if entry.bytes+len(entry.json) > projectionBytes {
			c.bytes -= entry.bytes
			delete(c.values, cursor)
		} else {
			for c.bytes+len(entry.json) > projectionBytes {
				c.evictLocked(cursor)
			}
			c.bytes += len(entry.json)
			entry.bytes += len(entry.json)
			entry.jsonCharged = true
		}
	}
	c.mu.Unlock()
	if !current {
		value = proto.Clone(value).(*v1.Notification)
		a.sanitize(value)
		return protojson.Marshal(value)
	}
	return entry.json, entry.err
}
