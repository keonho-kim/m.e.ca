// Purpose: Compose credential persistence with one live process-owned secret registry.
// Pattern: Explicit resource aggregate with a zero-value redaction registry.
// Usage: Composition roots supply Vault to model changes and encrypted key maintenance.
// Related: internal/redaction/redaction.go, internal/config/credentials/keyring.go.
package credentials

import "github.com/keonho-kim/meca/internal/redaction"

type Vault struct {
	State string
	Keys  KeyStore
	redaction.Registry
}
