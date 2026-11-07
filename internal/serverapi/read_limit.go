// Purpose: Share one original-restoration slot across native and compatible server reads.
// Pattern: Process-scoped work admission.
// Usage: API.Call and OpenAI result writers hold the slot until large output is released.
// Related: internal/serverapi/commands.go, internal/serverapi/openai/result.go.
package serverapi

import "github.com/keonho-kim/meca/internal/serverapi/admission"

func (a *API) readLimit() *admission.Budget {
	a.readOnce.Do(func() { a.reads = admission.New(1) })
	return a.reads
}
