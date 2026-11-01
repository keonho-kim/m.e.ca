// Purpose: Keep optional LAT planning, review and memory state under one request-owned value.
// Pattern: Feature state composed into execution without a separate loop or worker.
// Usage: LAT phases mutate reviewState; normal execution observes only its explicit boundaries.
// Related: internal/agent/run/lat.go, internal/agent/run/lat_memory.go, internal/agent/run/steering.go.
package run

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type reviewState struct {
	memory     *reviewMemory
	generator  modelapi.Generator
	steering   []string
	plan       string
	observed   bool
	reviewed   bool
	draftSaved bool
	active     bool
}

func (r *reviewState) invalidatePlan() {
	r.plan = ""
	r.reviewed = false
	r.observed = false
}
