// Purpose: Own local credential-key persistence and installation diagnostics.
// Pattern: Central diagnostic catalog.
// Usage: The local key adapter reports private storage and verification failures.
// Related: internal/storage/credentialkeys/store.go, internal/storage/credentialstore/crypto.go.
package catalog

const (
	LocalKeysInvalidKey         Message = "invalid local encryption key or reference"
	LocalKeysConflict           Message = "encryption key reference already contains different material"
	LocalKeysPrivateDirectory   Message = "credential-keys requires a real directory owned by the current user with permissions 0700"
	LocalKeysPrivateFile        Message = "credential-keys/keys.db requires a regular file owned by the current user with permissions 0600"
	LocalKeysVerificationFailed Message = "local credential key verification failed"
	LocalKeysUnsupportedStore   Message = "unsupported local encryption-key database"
	LocalKeysMissing            Message = "local encryption key was not found. Restore credentials.db and credential-keys/keys.db together from a backup (%w)"
	LocalKeysAccessDenied       Message = "access to credential-keys/keys.db was denied. Check ownership and permissions 0700 for its directory and 0600 for the file (%w)"
	LocalKeysReadFailed         Message = "local encryption-key storage could not be read (%w)"
	LocalKeysWriteFailed        Message = "local encryption-key storage could not be written. Check disk space, permissions and filesystem support (%w)"
)
