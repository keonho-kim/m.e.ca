// Purpose: Bound temporary command archives and reclaim only results acknowledged as durable.
// Pattern: Executor-owned serialized filesystem budget with completed-result eviction.
// Usage: Live projections and final archives write here; AcknowledgeCommand makes them reclaimable.
// Related: internal/tools/local/command_live.go, internal/tools/local/command_archive.go, internal/tools/local/executor.go.
package local

import (
	"container/list"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/keonho-kim/meca/internal/storage/capacity"
)

const (
	maxArtifactBytes    = 256 << 20
	maxArtifactCommands = 256
)

type commandArtifact struct {
	id      string
	files   map[string]int64
	retired *list.Element
}

type artifactRetention struct {
	mu            sync.Mutex
	entries       map[string]*commandArtifact
	retired       list.List
	bytes         int64
	limitBytes    int64
	limitCommands int
}

func (r *artifactRetention) acknowledge(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry := r.entries[id]; entry != nil && entry.retired == nil {
		entry.retired = r.retired.PushBack(entry)
	}
}

func (r *artifactRetention) evict() error {
	oldest := r.retired.Front()
	if oldest == nil {
		return capacity.ErrLimit
	}
	entry := oldest.Value.(*commandArtifact)
	for path, size := range entry.files {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		delete(entry.files, path)
		r.bytes -= size
	}
	delete(r.entries, entry.id)
	r.retired.Remove(oldest)
	return nil
}

func (e *Executor) writeArtifact(id, path string, body []byte) error {
	r := &e.retained
	r.mu.Lock()
	defer r.mu.Unlock()
	limitBytes, limitCount := r.limitBytes, r.limitCommands
	if limitBytes == 0 {
		limitBytes = maxArtifactBytes
	}
	if limitCount == 0 {
		limitCount = maxArtifactCommands
	}
	if int64(len(body)) > limitBytes {
		return capacity.ErrLimit
	}
	entry := r.entries[id]
	if entry != nil && entry.retired != nil {
		return fmt.Errorf("completed command artifacts are immutable")
	}
	var previous int64
	if entry != nil {
		previous = entry.files[path]
	}
	delta := int64(len(body)) - previous
	for r.bytes+delta > limitBytes || entry == nil && len(r.entries) >= limitCount {
		if err := r.evict(); err != nil {
			return err
		}
	}
	if len(body) > 0 {
		for {
			err := capacity.CheckFree(e.artifacts, 1<<30+uint64(len(body)))
			if err == nil {
				break
			}
			if !errors.Is(err, capacity.ErrLimit) || r.retired.Len() == 0 {
				return err
			}
			if err := r.evict(); err != nil {
				return err
			}
		}
	}
	if err := writeArtifactFile(path, body); err != nil {
		return err
	}
	if entry == nil {
		if r.entries == nil {
			r.entries = make(map[string]*commandArtifact)
		}
		entry = &commandArtifact{id: id, files: make(map[string]int64)}
		r.entries[id] = entry
	}
	entry.files[path] = int64(len(body))
	r.bytes += delta
	return nil
}

func (r *artifactRetention) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = nil
	r.retired.Init()
	r.bytes = 0
}

// Temporary logs need atomic replacement, not another durable fsync boundary.
func writeArtifactFile(path string, body []byte) (resultErr error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".pending-log-*")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(file.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, err)
		}
	}()
	_, writeErr := file.Write(body)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
