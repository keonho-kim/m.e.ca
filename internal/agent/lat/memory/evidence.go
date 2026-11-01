// Purpose: Share one bounded immutable evidence body during LAT preparation only.
// Pattern: Caller-owned short-lived snapshot with explicit invalidation keys.
// Usage: Preparation passes Evidence to Tail and clears it before implementation.
// Related: internal/agent/lat/memory/controller.go, internal/agent/run/lat_prepare.go.
package memory

// Evidence owns at most 64 KiB of already-selected JSON, never a database or model history.
// Returned bodies share its immutable bytes; callers must not modify them.
type Evidence struct {
	body                []byte
	current             string
	through             int64
	revision, redaction int
}

func (e *Evidence) Clear() {
	if e != nil {
		*e = Evidence{}
	}
}

func (e *Evidence) get(state State, mark Mark, current string, redaction, budget int) []byte {
	if e != nil && e.revision == state.Revision && e.through == mark.Through && e.current == current && e.redaction == redaction && len(e.body) <= budget {
		return e.body
	}
	return nil
}

func (e *Evidence) set(state State, mark Mark, current string, redaction int, body []byte) {
	if e != nil && len(body) <= 64*1024 {
		*e = Evidence{body: body, current: current, through: mark.Through, revision: state.Revision, redaction: redaction}
	}
}
