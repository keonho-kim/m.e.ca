// Purpose: Bound pending uploaded images across every project in one server process.
// Pattern: Explicit shared resource allowance.
// Usage: StageImage reserves until release, expiry or workspace shutdown.
// Related: internal/commands/server/attachments.go, cmd/meca/server.go.
package server

import "sync"

type AttachmentPool struct {
	mu           sync.Mutex
	count, bytes int
}

func NewAttachmentPool() *AttachmentPool { return &AttachmentPool{} }
func (p *AttachmentPool) reserve(bytes int) (func(), bool) {
	if p == nil {
		return func() {}, true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.count >= 16 || bytes > attachmentBytes-p.bytes {
		return nil, false
	}
	p.count++
	p.bytes += bytes
	return sync.OnceFunc(func() { p.mu.Lock(); p.count--; p.bytes -= bytes; p.mu.Unlock() }), true
}
