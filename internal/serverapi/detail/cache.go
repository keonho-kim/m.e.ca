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
)

const memoryBytes = 64 << 10

type Request struct {
	Session string
	Source  server.DetailSource
	Version func() int
	Redact  func(string) string
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
		return nil, 0, &server.Error{Code: "invalid_argument", Message: "detail offset must be nonnegative and limit must be 1 to 65536"}
	}
	select {
	case c.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}
	defer func() { <-c.gate }()
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		version := request.Version()
		if c.loaded && c.session == request.Session && c.source == request.Source && c.version == version {
			break
		}
		original, err := request.Load()
		if err != nil {
			return nil, 0, err
		}
		body, err := marshalDocument(original, request.Redact)
		if err != nil {
			return nil, 0, err
		}
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		if version != request.Version() {
			continue
		}
		if err := c.replace(ctx, request, version, body); err != nil {
			if cause := ctx.Err(); cause != nil {
				return nil, 0, cause
			}
			if version != request.Version() {
				continue
			}
			// A spool failure must not hide a successfully restored public original.
			return memoryPage(body, offset, limit)
		}
	}
	if c.file == nil {
		return memoryPage(c.memory, offset, limit)
	}
	if offset > c.size {
		return nil, 0, &server.Error{Code: "invalid_argument", Message: "detail offset exceeds original"}
	}
	end := min(c.size, offset+int64(limit))
	page := make([]byte, int(end-offset))
	_, err := c.file.ReadAt(page, offset)
	return page, c.size, err
}

func memoryPage(body []byte, offset int64, limit int) ([]byte, int64, error) {
	if offset > int64(len(body)) {
		return nil, 0, &server.Error{Code: "invalid_argument", Message: "detail offset exceeds original"}
	}
	end := min(int64(len(body)), offset+int64(limit))
	return append([]byte(nil), body[offset:end]...), int64(len(body)), nil
}

func (c *Cache) replace(ctx context.Context, request Request, version int, body []byte) error {
	if err := c.clear(); err != nil {
		return err
	}
	if len(body) <= memoryBytes {
		c.memory = body
	} else {
		file, err := os.CreateTemp("", "meca-server-detail-*")
		if err != nil {
			return err
		}
		if err := os.Remove(file.Name()); err != nil {
			return errors.Join(err, file.Close())
		}
		for rest := body; len(rest) > 0; {
			if err := ctx.Err(); err != nil {
				return errors.Join(err, file.Close())
			}
			n := min(len(rest), memoryBytes)
			if _, err := file.Write(rest[:n]); err != nil {
				return errors.Join(err, file.Close())
			}
			rest = rest[n:]
		}
		c.file = file
	}
	c.session, c.source, c.version, c.size, c.loaded = request.Session, request.Source, version, int64(len(body)), true
	return nil
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
