// Purpose: Reuse sanitized live excerpts and logs until a command's captured output changes.
// Pattern: Executor-owned versioned projection with bounded excerpts.
// Usage: Running command observations request only changed streams; acknowledgement releases cache entries.
// Related: internal/tools/local/commands.go, internal/tools/local/process/projection_read.go, internal/tools/local/output.go.
package local

import (
	"sync"

	"github.com/keonho-kim/meca/internal/tools/local/process"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

type liveCapture struct {
	revision       uint64
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
	previous, finalized := c.streams, c.finalized
	c.mu.Unlock()
	if finalized {
		return previous, nil
	}
	after := [2]uint64{previous[0].revision, previous[1].revision}
	version := e.commandVersion()
	for i := range previous {
		if previous[i].version != version {
			after[i] = 0
		}
	}
	updates, release, err := e.commands.ReadProjection(id, after)
	if err != nil {
		if release != nil {
			release()
		}
		return previous, err
	}
	defer release()
	if !c.mu.TryLock() {
		return previous, process.ErrProjectionBusy
	}
	defer c.mu.Unlock()
	if c.finalized {
		return c.streams, nil
	}
	previous = c.streams
	for i, stream := range []string{"stdout", "stderr"} {
		if updates[i].Revision < previous[i].revision || updates[i].Revision == previous[i].revision && previous[i].version == version {
			continue
		}
		previous[i], err = e.projectLiveCapture(id, stream, previous[i], updates[i])
		if err != nil {
			return previous, err
		}
		c.streams[i] = previous[i]
	}
	c.streams = previous
	return previous, nil
}

func (e *Executor) projectLiveCapture(id, stream string, previous liveCapture, update process.Capture) (liveCapture, error) {
	version := e.commandVersion()
	if previous.revision == update.Revision && previous.version == version {
		return previous, nil
	}
	text := textformat.RedactSource(update.Text, e.redact)
	next := liveCapture{revision: update.Revision, excerpt: commandExcerpt(text), bytes: len(text), version: version}
	if len(text) > outputExcerptBytes {
		path, err := e.commandPath(id, "."+stream+".log")
		if err != nil {
			return previous, err
		}
		if err := e.writeArtifact(id, path, []byte(text)); err != nil {
			return previous, err
		}
		next.path = path
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
