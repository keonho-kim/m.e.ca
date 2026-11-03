// Purpose: Track interleaved live blocks independently of their position in transcript history.
// Pattern: Event-loop-owned bounded identity index.
// Usage: View mutations register streams; Transcript includes them in the shared display budget.
// Related: internal/tui/viewport/transcript_index.go, internal/tui/viewport/cache_budget.go.
package viewport

import (
	"cmp"
	"slices"
	"sort"
)

func (v *View) trackStream(id int, active bool) {
	if !active {
		delete(v.streams, id)
		return
	}
	if v.streams == nil {
		v.streams = map[int]bool{}
	}
	v.streams[id] = true
}

func (v *View) FinishStream(id int) {
	i := sort.Search(len(v.Blocks), func(i int) bool { return v.Blocks[i].ID >= id })
	if i < len(v.Blocks) && v.Blocks[i].ID == id {
		v.Blocks[i].Streaming, v.Blocks[i].live = false, false
		v.Blocks[i].releaseParsed()
	}
	delete(v.streams, id)
}

func (t *Transcript) activeStreams(v *View) {
	clear(t.active)
	t.active = t.active[:0]
	for id := range v.streams {
		i := sort.Search(len(v.Blocks), func(i int) bool { return v.Blocks[i].ID >= id })
		if i < len(v.Blocks) && v.Blocks[i].ID == id {
			t.active = append(t.active, &v.Blocks[i])
		}
	}
	slices.SortFunc(t.active, func(a, b *Block) int { return cmp.Compare(a.ID, b.ID) })
}
