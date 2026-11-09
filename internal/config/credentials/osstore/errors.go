// Purpose: Define OS credential-store absence independently of a concrete platform.
// Pattern: Adapter error contract.
// Usage: Vault composes native OS key storage with the encrypted credential repository.
// Related: internal/config/credentials/osstore/store_darwin.go, internal/config/credentials/osstore/store_linux.go.
package osstore

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var ErrNotFound = faults.Errorf(diagcatalog.OsstoreOSCredentialWasNotFound)
