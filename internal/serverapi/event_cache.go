// Purpose: Share immutable masked event projections and JSON across simultaneous observers.
// Pattern: Observer-lifetime bounded cache keyed by replay cursor and credential revision.
// Usage: Every transport projects via projectEvent; SSE reuses encoded JSON when available.
// Related: internal/serverapi/watch.go, internal/serverapi/http.go, internal/serverapi/projection.go.
package serverapi

import (
	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"sync"
)

type projectedEvent struct {
	value *v1.Notification
	json  []byte
	order uint64
	bytes int
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
	defer c.mu.Unlock()
	version := 0
	if a.RedactionVersion != nil {
		version = a.RedactionVersion()
	}
	if c.version != version {
		c.values = nil
		c.bytes = 0
		c.version = version
	}
	cache := c.observers > 1 && a.RedactionVersion != nil && value.Cursor != ""
	if cache {
		if hit := c.values[value.Cursor]; hit != nil {
			return hit.value
		}
	}
	notification := &v1.Notification{Body: &v1.Notification_Event{Event: event(value)}}
	a.sanitize(notification)
	if !cache || version != a.RedactionVersion() {
		return notification
	}
	size := proto.Size(notification) + 1024
	if size > 1<<20 {
		return notification
	}
	for len(c.values) >= 128 || c.bytes+size > 1<<20 {
		var oldest string
		for key, item := range c.values {
			if oldest == "" || item.order < c.values[oldest].order {
				oldest = key
			}
		}
		c.bytes -= c.values[oldest].bytes
		delete(c.values, oldest)
	}
	if c.values == nil {
		c.values = make(map[string]*projectedEvent)
	}
	c.order++
	c.values[value.Cursor] = &projectedEvent{value: notification, order: c.order, bytes: size}
	c.bytes += size
	return notification
}
func (a *API) notificationJSON(value *v1.Notification) ([]byte, error) {
	c := &a.events
	c.mu.Lock()
	defer c.mu.Unlock()
	version := 0
	if a.RedactionVersion != nil {
		version = a.RedactionVersion()
	}
	if c.version == version {
		if hit := c.values[notificationCursor(value)]; hit != nil && hit.value == value {
			if hit.json != nil {
				return hit.json, nil
			}
			body, err := protojson.Marshal(value)
			if err == nil && c.bytes+len(body) <= 1<<20 {
				hit.json = body
				hit.bytes += len(body)
				c.bytes += len(body)
			}
			return body, err
		}
	}
	if c.version != version {
		value = proto.Clone(value).(*v1.Notification)
		a.sanitize(value)
	}
	return protojson.Marshal(value)
}
