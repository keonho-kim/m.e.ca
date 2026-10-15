// Purpose: Bound whole-file source reads used for mutation and content search.
// Pattern: Simple Module.
// Usage: Local version snapshots, source reads use MaxBytes.
// Related: internal/tools/local/paths.go, internal/tools/local/file/read.go.
package file

const MaxBytes = 16 * 1024 * 1024
