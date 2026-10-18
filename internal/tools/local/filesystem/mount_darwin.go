//go:build darwin

// Purpose: Use Darwin device identities for mount boundaries.
// Pattern: Platform policy Adapter.
// Usage: checkTarget already compares st_dev for existing parent components.
// Related: internal/tools/local/filesystem/target_unix.go.
package filesystem

func checkMount(string, string) error { return nil }
