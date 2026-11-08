// Purpose: Page one masked public JSON document without reloading its original for every request.
// Pattern: Session-scoped single-entry cache with bounded memory and an unlinked private spool.
// Usage: API detail reads call Read; the conversation's composed cleanup calls CloseSession.
// Related: internal/serverapi/detail/document.go, internal/commands/server/detail.go.
package detail

import (
	"context"
	"errors"
	"os"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

const memoryBytes = 64 << 10

type Request struct {
	Session string
	Source  server.DetailSource
	Version func() int
	NewMask func() func(string, bool) string
	Load    func() (server.DetailOriginal, error)
}

type Cache struct {
	gate    chan struct{}
	session string
	source  server.DetailSource
	version int
	loaded  bool
	memory  []byte
	file    *os.File
	size    int64
}

func NewCache() *Cache { return &Cache{gate: make(chan struct{}, 1)} }

func (c *Cache) Read(ctx context.Context, request Request, offset int64, limit int) ([]byte, int64, error) {
	if offset < 0 || limit < 1 || limit > 64<<10 {
		return nil, 0, faults.New(diagcatalog.InvalidArgument, diagcatalog.DetailDetailOffsetMustBeNonnegativeAndLimit)
	}
	select {
	case c.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}
	defer func() { <-c.gate }()
	for range 2 {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		version := request.Version()
		if c.loaded && c.session == request.Session && c.source == request.Source && c.version == version {
			break
		}
		if err := c.clear(); err != nil {
			return nil, 0, err
		}
		spool := newDocumentSpool(offset, limit)
		if err := loadDocument(ctx, spool, request); err != nil {
			return nil, 0, errors.Join(err, spool.close())
		}
		if spool.failure != nil {
			diagnostics.Log(ctx, diagcatalog.DetailCacheBypassed, "error_code", faults.Code(spool.failure))
		}
		if spool.cleanupErr != nil {
			return nil, 0, spool.close()
		}
		if err := ctx.Err(); err != nil {
			return nil, 0, errors.Join(err, spool.close())
		}
		if version != request.Version() {
			if err := spool.close(); err != nil {
				return nil, 0, err
			}
			continue
		}
		if offset > spool.total {
			return nil, 0, errors.Join(faults.New(diagcatalog.InvalidArgument, diagcatalog.DetailDetailOffsetExceedsOriginal), spool.close())
		}
		if spool.cacheable {
			c.file, c.memory, c.size = spool.file, spool.memory, spool.total
			spool.file = nil
			c.session, c.source, c.version, c.loaded = request.Session, request.Source, version, true
		}
		return spool.page, spool.total, nil
	}
	if !c.loaded || c.session != request.Session || c.source != request.Source || c.version != request.Version() {
		return nil, 0, faults.Errorf(diagcatalog.DetailMaskChanged)
	}
	if c.file == nil {
		return memoryPage(c.memory, offset, limit)
	}
	if offset > c.size {
		return nil, 0, faults.New(diagcatalog.InvalidArgument, diagcatalog.DetailDetailOffsetExceedsOriginal)
	}
	end := min(c.size, offset+int64(limit))
	page := make([]byte, int(end-offset))
	_, err := c.file.ReadAt(page, offset)
	return page, c.size, err
}

// The original is transient. Keep its reservation until the masked document
// has reached the bounded cache or private spool, then discard all references.
func loadDocument(ctx context.Context, spool *documentSpool, request Request) error {
	release, err := processmemory.AcquirePayload(ctx, server.MaxOriginalBytes)
	if err != nil {
		return err
	}
	defer release()
	original, err := request.Load()
	if err != nil {
		return err
	}
	return writeDocument(ctx, spool, original, request.NewMask)
}

func memoryPage(body []byte, offset int64, limit int) ([]byte, int64, error) {
	if offset > int64(len(body)) {
		return nil, 0, faults.New(diagcatalog.InvalidArgument, diagcatalog.DetailDetailOffsetExceedsOriginal)
	}
	end := min(int64(len(body)), offset+int64(limit))
	return append([]byte(nil), body[offset:end]...), int64(len(body)), nil
}

func (c *Cache) clear() error {
	file := c.file
	c.file, c.memory, c.loaded, c.size, c.session = nil, nil, false, 0, ""
	if file != nil {
		return file.Close()
	}
	return nil
}

func (c *Cache) CloseSession(session string) error {
	c.gate <- struct{}{}
	defer func() { <-c.gate }()
	if c.session != session {
		return nil
	}
	return c.clear()
}

func (c *Cache) Close() error {
	c.gate <- struct{}{}
	defer func() { <-c.gate }()
	return c.clear()
}
