// Purpose: Retain small detail documents or a private spool while always collecting the requested page.
// Pattern: Bounded output sink with a no-storage fallback.
// Usage: Cold detail serialization writes once; spool failure preserves paged reads without full buffering.
// Related: internal/serverapi/detail/cache.go, internal/serverapi/detail/write.go.
package detail

import (
	"errors"

	"github.com/keonho-kim/meca/internal/storage/capacity"
	"io"
	"os"
)

const maxSpoolBytes = 64 << 20

type documentSpool struct {
	archive       *chunkArchive
	budget        *spoolBudget
	memory, page  []byte
	file          *os.File
	total, offset int64
	limit         int
	cacheable     bool
	failure       error
	cleanupErr    error
}

func newDocumentSpool(offset int64, limit int) *documentSpool {
	return &documentSpool{offset: offset, limit: limit, cacheable: true}
}

func (s *documentSpool) Write(body []byte) (int, error) {
	start, end := max(s.offset, s.total), min(s.offset+int64(s.limit), s.total+int64(len(body)))
	if start < end {
		s.page = append(s.page, body[start-s.total:end-s.total]...)
	}
	s.total += int64(len(body))
	if !s.cacheable {
		return len(body), nil
	}
	if s.file == nil && len(s.memory)+len(body) <= memoryBytes {
		s.memory = append(s.memory, body...)
		return len(body), nil
	}
	if s.file == nil {
		if err := capacity.CheckFree(os.TempDir(), (1<<30)+maxSpoolBytes); err != nil {
			s.disable(err)
			return len(body), nil
		}
		file, err := os.CreateTemp("", "meca-server-detail-*")
		if err != nil {
			s.disable(err)
			return len(body), nil
		}
		if err := os.Remove(file.Name()); err != nil {
			s.cleanupErr = errors.Join(file.Close(), os.Remove(file.Name()))
			s.disable(err)
			return len(body), nil
		}
		s.file = file
		s.archive = &chunkArchive{file: file, budget: s.budget}
		if n, err := s.archive.Write(s.memory); err != nil || n != len(s.memory) {
			if err == nil {
				err = io.ErrShortWrite
			}
			s.disable(err)
		}
		s.memory = nil
	}
	if s.cacheable {
		if n, err := s.archive.Write(body); err != nil || n != len(body) {
			if err == nil {
				err = io.ErrShortWrite
			}
			s.disable(err)
		}
	}
	return len(body), nil
}

func (s *documentSpool) disable(cause error) {
	s.failure = errors.Join(s.failure, cause)
	if s.archive != nil {
		s.archive.release()
		s.archive = nil
	}
	if s.file != nil {
		s.cleanupErr = errors.Join(s.cleanupErr, s.file.Close())
		s.file = nil
	}
	s.memory = nil
	s.cacheable = false
}

func (s *documentSpool) close() error {
	if s.archive != nil {
		s.archive.release()
		s.archive = nil
	}
	if s.file != nil {
		file := s.file
		s.file = nil
		return errors.Join(s.cleanupErr, file.Close())
	}
	return s.cleanupErr
}

var _ io.Writer = (*documentSpool)(nil)

func (s *documentSpool) finish() error {
	if s.archive != nil {
		return s.archive.flush()
	}
	return nil
}
