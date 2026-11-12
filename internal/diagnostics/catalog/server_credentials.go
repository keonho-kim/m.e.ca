// Purpose: Own public credential-publication and browser-session diagnostics.
// Pattern: Central diagnostic catalog.
// Usage: Server startup and credential file ownership use these messages.
// Related: internal/serverapi/token_lease_unix.go, cmd/meca/server.go.
package catalog

const (
	ServerTokenLockFailed     Message = "cannot lock the token file; use a filesystem that supports file locking: %w"
	ServerTokenPublisherOwned Message = "another server owns this token file; choose a different --token-file"
	ServerTokenLeasePrivate   Message = "the token file lease must be a private regular file owned by the current user"
)
