// Purpose: Query command presence and immutable completion metadata without live-output snapshots.
// Pattern: Bounded caller-owned projection under the registry lock.
// Usage: The executor collects terminal results; runtime presence checks include unacknowledged jobs.
// Related: internal/tools/local/process/command_finish.go, internal/tools/local/commands.go.
package process

import (
	"cmp"
	"slices"
)

// Completion contains only fields needed to locate and commit an archived result.
type Completion struct {
	ID, CallID, Command, Error string
}

// HasCommands includes running jobs and terminal results not yet acknowledged.
func (r *Registry) HasCommands() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.jobs) > 0
}

// AppendCompleted reuses caller storage and sorts only the appended suffix.
// Completion closes after the last state write; terminal fields are immutable.
func (r *Registry) AppendCompleted(dst []Completion) []Completion {
	r.mu.Lock()
	defer r.mu.Unlock()
	from := len(dst)
	for _, j := range r.jobs {
		select {
		case <-j.done:
			state := &j.state
			dst = append(dst, Completion{state.ID, state.CallID, state.Command, state.Error})
		default:
		}
	}
	slices.SortFunc(dst[from:], func(a, b Completion) int { return cmp.Compare(a.ID, b.ID) })
	return dst
}
