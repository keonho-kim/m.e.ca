// Purpose: Name the two supported executable distributions.
// Pattern: Compile-time distribution contract.
// Usage: Startup, help and release selection use these names without runtime configuration.
// Related: internal/distribution/full.go, internal/distribution/minimum.go.
package distribution

const (
	Minimum = "minimum"
	Full    = "full"
)
