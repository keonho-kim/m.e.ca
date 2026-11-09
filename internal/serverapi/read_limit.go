// Purpose: Share one restoration slot across native history, detail and skill reads.
// Pattern: Process-scoped work admission.
// Usage: API.Call holds admission until the selected read operation returns.
// Related: internal/serverapi/commands.go, internal/serverapi/detail.go.
package serverapi

import "github.com/keonho-kim/meca/internal/serverapi/admission"

func (a *API) readLimit() *admission.Budget {
	a.readOnce.Do(func() { a.reads = admission.New(1) })
	return a.reads
}

func (a *API) browseLimit() *admission.Budget {
	a.browseOnce.Do(func() { a.browse = admission.New(4) })
	return a.browse
}
