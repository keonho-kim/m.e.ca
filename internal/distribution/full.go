//go:build !minimum

// Purpose: Select the full executable for ordinary source builds.
// Pattern: Build constraint selection.
// Usage: Go includes this constant when the minimum tag is absent.
// Related: internal/distribution/variants.go, .goreleaser.yaml.
package distribution

const Name = Full
