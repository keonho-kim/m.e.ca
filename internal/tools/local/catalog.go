// Purpose: Expose the immutable native catalog through the local executor contract.
// Pattern: Tool executor adapter.
// Usage: Runtime calls Definitions before requesting native model tool calls.
// Related: internal/tools/local/catalog/catalog.go, internal/tools/local/executor.go.
package local

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
)

func (*Executor) Definitions() []toolapi.Definition { return toolcatalog.Definitions() }
