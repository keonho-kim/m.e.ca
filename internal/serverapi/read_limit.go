// Purpose: Weight native read work independently of execution and retained-byte budgets.
// Pattern: Process-scoped work admission.
// Usage: API.Call holds admission until the selected read operation returns.
// Related: internal/serverapi/commands.go, internal/serverapi/detail.go.
package serverapi

import "github.com/keonho-kim/meca/internal/serverapi/admission"

const readCapacity = 8

func (a *API) readLimit() *admission.Budget {
	a.readOnce.Do(func() {
		if a.reads == nil {
			a.reads = admission.New(readCapacity)
		}
	})
	return a.reads
}

func (a *API) browseLimit() *admission.Budget {
	a.browseOnce.Do(func() {
		if a.browse == nil {
			a.browse = admission.New(4)
		}
	})
	return a.browse
}
