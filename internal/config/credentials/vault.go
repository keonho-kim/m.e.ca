// Purpose: Compose credential persistence with one live process-owned secret registry.
// Pattern: Explicit resource aggregate with a zero-value redaction registry.
// Usage: Composition roots supply Vault to model changes and encrypted key maintenance.
// Related: internal/redaction/redaction.go, internal/config/credentials/persistence.go.
package credentials

import (
	"github.com/keonho-kim/meca/internal/redaction"
	"github.com/keonho-kim/meca/internal/storage/credentialstore"
)

type Vault struct {
	State string
	Keys  credentialstore.Keys
	redaction.Registry
}
