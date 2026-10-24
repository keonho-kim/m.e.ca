// Purpose: Represent valid active-worker and cancellation states without independent boolean flags.
// Pattern: Explicit state machine owned by the terminal event loop.
// Usage: Task, setup and diagnostic entry points start work; cancellation and completion transition it.
// Related: internal/tui/session/tasks.go, internal/tui/session/worker_result.go, internal/tui/session/activity.go.
package session

type workerState uint8

const (
	workerIdle workerState = iota
	workerAgent
	workerAgentStopping
	workerSetup
	workerSetupStopping
)

func (s workerState) active() bool { return s != workerIdle }
func (s workerState) agent() bool  { return s == workerAgent || s == workerAgentStopping }
func (s workerState) stopping() bool {
	return s == workerAgentStopping || s == workerSetupStopping
}

func (s *workerState) stop() {
	switch *s {
	case workerAgent:
		*s = workerAgentStopping
	case workerSetup:
		*s = workerSetupStopping
	}
}
