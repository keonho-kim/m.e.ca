// Purpose: Reuse sanitized live excerpts and logs until a command's captured output changes.
// Pattern: Executor-owned versioned projection with bounded excerpts.
// Usage: Running command observations request only changed streams; acknowledgement releases cache entries.
// Related: internal/tools/local/commands.go, internal/tools/local/process/projection_read.go, internal/tools/local/output.go.
package local

import (
	"errors"
	"sync"

	"github.com/keonho-kim/meca/internal/tools/local/process"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

type liveCapture struct {
	revision       uint64
	rawBytes       int64
	incremental    bool
	excerpt, path  string
	bytes, version int
}

type liveCaptures struct {
	mu     sync.Mutex
	values map[string]*liveCommandCapture
}

type liveCommandCapture struct {
	mu             sync.Mutex
	streams        [2]liveCapture
	masks          [2]func(string, bool) string
	finalized      bool
	captureRelease func()
	commandMu      sync.Mutex
	command        string
	commandVersion int
	commandCached  bool
}

func (c *liveCaptures) command(id string) *liveCommandCapture {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.values == nil {
		c.values = make(map[string]*liveCommandCapture)
	}
	if c.values[id] == nil {
		c.values[id] = &liveCommandCapture{}
	}
	return c.values[id]
}

func (c *liveCaptures) forget(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry := c.values[id]; entry != nil {
		entry.mu.Lock()
		if entry.captureRelease != nil {
			entry.captureRelease()
			entry.captureRelease = nil
		}
		entry.mu.Unlock()
	}
	delete(c.values, id)
}

func (c *liveCaptures) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, entry := range c.values {
		entry.mu.Lock()
		if entry.captureRelease != nil {
			entry.captureRelease()
			entry.captureRelease = nil
		}
		entry.mu.Unlock()
	}
	c.values = nil
}

func (e *Executor) commandCaptures(id string) ([2]liveCapture, error) {
	c := e.liveOutputs.command(id)
	if !c.mu.TryLock() {
		return [2]liveCapture{}, process.ErrProjectionBusy
	}
	defer c.mu.Unlock()
	if c.finalized {
		return c.streams, nil
	}
	for range 2 {
		version := e.commandVersion()
		var after [2]uint64
		var positions [2]int64
		for i, previous := range c.streams {
			if previous.version == version && (e.newChunkMask == nil || previous.incremental && c.masks[i] != nil) {
				after[i] = previous.revision
				if e.newChunkMask != nil {
					positions[i] = previous.rawBytes
				}
			}
		}
		updates, release, err := e.commands.ReadProjection(id, after, positions)
		if err != nil {
			if release != nil {
				release()
			}
			return [2]liveCapture{}, err
		}
		err = e.applyCaptures(id, c, updates, version)
		release()
		if err == nil {
			return c.streams, nil
		}
		// A changed matcher requires fresh full captures, never reapplication of a
		// delta produced from an earlier byte position or partially advanced mask.
		if !errors.Is(err, process.ErrProjectionBusy) {
			return [2]liveCapture{}, err
		}
	}
	return [2]liveCapture{}, process.ErrProjectionBusy
}

func (e *Executor) applyCaptures(id string, c *liveCommandCapture, updates [2]process.Capture, version int) error {
	if e.commandVersion() != version {
		c.masks = [2]func(string, bool) string{}
		return process.ErrProjectionBusy
	}
	for i, stream := range []string{"stdout", "stderr"} {
		previous := c.streams[i]
		if updates[i].Revision == previous.revision && previous.version == version {
			continue
		}
		var next liveCapture
		var err error
		if e.newChunkMask != nil && (updates[i].Append || updates[i].Total <= 16<<20) {
			next, err = e.projectIncrementalCapture(id, stream, previous, updates[i], &c.masks[i])
		} else {
			next, err = e.projectLiveCapture(id, stream, previous, updates[i])
			next.rawBytes = updates[i].Total
		}
		if err != nil {
			c.streams[i] = liveCapture{}
			c.masks[i] = nil
			return err
		}
		c.streams[i] = next
	}
	if e.commandVersion() != version {
		c.masks = [2]func(string, bool) string{}
		c.streams = [2]liveCapture{}
		return process.ErrProjectionBusy
	}
	return nil
}

func (e *Executor) projectLiveCapture(id, stream string, previous liveCapture, update process.Capture) (liveCapture, error) {
	version := e.commandVersion()
	if previous.revision == update.Revision && previous.version == version {
		return previous, nil
	}
	text := textformat.RedactSource(update.Text, e.redact)
	next := liveCapture{revision: update.Revision, excerpt: commandExcerpt(text), bytes: len(text), version: version}
	var err error
	next.path, err = e.projectCaptureFile(id, stream, text)
	if err != nil {
		return previous, err
	}
	return next, nil
}

func (c *liveCaptures) releaseProjections() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, entry := range c.values {
		entry.mu.Lock()
		if entry.captureRelease != nil {
			entry.captureRelease()
			entry.captureRelease = nil
		}
		entry.mu.Unlock()
	}
}
