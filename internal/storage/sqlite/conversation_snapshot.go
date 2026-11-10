// Purpose: Retain bounded session ordering references for independent cursor traversals.
// Pattern: Catalog-owned immutable snapshots with lazy expiry and no database handles.
// Usage: ConversationPage scans once, then reads only the IDs needed for subsequent pages.
// Related: internal/storage/sqlite/conversation_page.go, internal/storage/session/session.go.
package sqlite

import (
	"crypto/rand"
	"slices"
	"sync"
	"time"
	"unsafe"

	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

const (
	conversationPageLifetime  = 2 * time.Minute
	conversationPageBytes     = 8 << 20
	conversationSnapshotBytes = 2 << 20
	conversationSnapshots     = 128
)

type conversationReference struct {
	id      string
	updated time.Time
}

type conversationSnapshot struct {
	project string
	refs    []conversationReference
	expires time.Time
	used    time.Time
}

func (p conversationSnapshot) bytes() int {
	size := 256 + len(p.project) + cap(p.refs)*int(unsafe.Sizeof(conversationReference{}))
	for _, ref := range p.refs {
		size += len(ref.id)
	}
	return size
}

type conversationPages struct {
	mu     sync.Mutex
	values map[string]conversationSnapshot
	now    func() time.Time
}

func (p *conversationPages) clock() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

func (p *conversationPages) prune(now time.Time) {
	for key, value := range p.values {
		if !now.Before(value.expires) {
			delete(p.values, key)
		}
	}
}

func (p *conversationPages) put(project string, refs []conversationReference) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.clock()
	p.prune(now)
	value := conversationSnapshot{project: project, refs: refs, expires: now.Add(conversationPageLifetime), used: now}
	total := value.bytes()
	for id, previous := range p.values {
		if previous.project == project && slices.EqualFunc(previous.refs, refs, func(a, b conversationReference) bool { return a.id == b.id && a.updated.Equal(b.updated) }) {
			previous.expires = now.Add(conversationPageLifetime)
			previous.used = now
			p.values[id] = previous
			return id, nil
		}
		total += previous.bytes()
	}
	if value.bytes() > conversationPageBytes {
		return "", sessionstate.ErrPageLimit
	}
	for len(p.values) >= conversationSnapshots || total > conversationPageBytes {
		var oldest string
		for id, previous := range p.values {
			if oldest == "" || previous.used.Before(p.values[oldest].used) {
				oldest = id
			}
		}
		total -= p.values[oldest].bytes()
		delete(p.values, oldest)
	}
	if p.values == nil {
		p.values = make(map[string]conversationSnapshot)
	}
	id := rand.Text()
	p.values[id] = value
	return id, nil
}

func (p *conversationPages) get(project, id string) ([]conversationReference, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.clock()
	p.prune(now)
	value, ok := p.values[id]
	if !ok || value.project != project {
		return nil, sessionstate.ErrCursor
	}
	value.used = now
	p.values[id] = value
	return value.refs, nil
}
