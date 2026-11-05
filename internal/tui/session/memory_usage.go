// Purpose: Cache one current-process memory label for the terminal header without persisting samples.
// Pattern: Event-loop-owned projection with a native-reader boundary.
// Usage: The adaptive status clock samples memory; every screen reuses the latest formatted value.
// Related: internal/processmemory/resident_darwin.go, internal/processmemory/resident_linux.go, internal/tui/view/memory.go.
package session

import (
	"time"

	"github.com/keonho-kim/meca/internal/processmemory"
	"github.com/keonho-kim/meca/internal/tui/view"
)

type memoryUsage struct {
	read  func() (uint64, error)
	value view.MemorySize
	label string
}

func memoryInterval(active bool) time.Duration {
	if active {
		return 250 * time.Millisecond
	}
	return time.Second
}

func (m *memoryUsage) sample() bool {
	read := m.read
	if read == nil {
		read = processmemory.ResidentBytes
	}
	value, err := read()
	if err != nil {
		value = 0
	}
	size := view.ResidentMemory(value)
	if size == m.value && m.label != "" {
		return false
	}
	m.value, m.label = size, size.Label()
	return true
}
