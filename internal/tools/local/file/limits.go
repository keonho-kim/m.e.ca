// Purpose: Bound whole-file source reads used for mutation and content search.
// Pattern: Simple Module.
// Usage: Local version snapshots, source reads and search use MaxBytes.
// Related: internal/tools/local/paths.go, internal/tools/local/search/search.go.
package file

const MaxBytes = 4 * 1024 * 1024
