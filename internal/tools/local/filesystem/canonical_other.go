//go:build !darwin

// Purpose: Retain resolved native paths on platforms without macOS case and firmlink normalization.
// Pattern: Platform path projection.
// Usage: Canonical already resolves symbolic links before calling physicalPath.
// Related: internal/tools/local/filesystem/paths.go, internal/tools/local/filesystem/canonical_darwin.go.
package filesystem

func physicalPath(path string) (string, error) { return path, nil }
