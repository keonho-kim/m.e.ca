// Purpose: Define OS credential-store absence independently of a concrete platform.
// Pattern: Adapter error contract.
// Usage: Vault composes native OS key storage with the encrypted credential repository.
// Related: internal/config/credentials/osstore/store_darwin.go, internal/config/credentials/osstore/store_linux.go.
package osstore

import "errors"

var ErrNotFound = errors.New("OS credential was not found")
