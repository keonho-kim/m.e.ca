// Purpose: Name the two supported executable distributions.
// Pattern: Distribution value contract.
// Usage: Startup selects upgrade targets; help and ordinary upgrades use the compiled Name.
// Related: internal/distribution/full.go, internal/distribution/tui.go.
package distribution

type Kind string

const (
	TUI  Kind = "tui"
	Full Kind = "full"
)
