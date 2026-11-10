//go:build minimum

// Purpose: Select the terminal-only executable at compilation.
// Pattern: Build constraint selection.
// Usage: Go includes this constant for the minimum release build.
// Related: internal/distribution/variants.go, .goreleaser.yaml.
package distribution

const Name = Minimum
