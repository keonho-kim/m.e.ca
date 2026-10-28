// Purpose: Reuse sanitized live excerpts and logs until a command's captured output changes.
// Pattern: Executor-owned versioned projection with bounded excerpts.
// Usage: Running command observations request only changed streams; acknowledgement releases cache entries.
// Related: internal/tools/local/commands.go, internal/tools/local/process/capture.go, internal/tools/local/output.go.
package local

import (
	"os"
	"sync"

	"github.com/keonho-kim/meca/internal/tools/local/process"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
)

type liveCapture struct {
	revision      uint64
	excerpt, path string
}

type liveCaptures struct {
	mu     sync.Mutex
	values map[string]*liveCommandCapture
}

type liveCommandCapture struct {
	mu      sync.Mutex
	streams [2]liveCapture
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
	delete(c.values, id)
}

func (c *liveCaptures) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values = nil
}

func (e *Executor) commandCaptures(id string) ([2]liveCapture, error) {
	c := e.liveOutputs.command(id)
	c.mu.Lock()
	defer c.mu.Unlock()
	previous := c.streams
	updates, err := e.commands.ReadOutput(id, [2]uint64{previous[0].revision, previous[1].revision})
	if err != nil {
		return previous, err
	}
	for i, stream := range []string{"stdout", "stderr"} {
		previous[i], err = e.projectLiveCapture(id, stream, previous[i], updates[i])
		if err != nil {
			return previous, err
		}
	}
	c.streams = previous
	return previous, nil
}

func (e *Executor) projectLiveCapture(id, stream string, previous liveCapture, update process.Capture) (liveCapture, error) {
	if previous.revision == update.Revision {
		return previous, nil
	}
	text := textformat.RedactSource(update.Text, e.redact)
	next := liveCapture{revision: update.Revision, excerpt: commandExcerpt(text)}
	if len(text) > outputExcerptBytes {
		path, err := e.commandPath(id, "."+stream+".live.log")
		if err != nil {
			return previous, err
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			return previous, err
		}
		next.path = path
	}
	return next, nil
}
