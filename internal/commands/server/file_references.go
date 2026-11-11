// Purpose: Define workspace-scoped reference persistence without restoring model runtimes.
// Pattern: Consumer-owned persistence boundary.
// Usage: File selections and workspace mutations operate on project references directly.
// Related: internal/storage/sqlite/project_references.go, internal/commands/server/workspace.go.
package server

import "context"

type FileReferences interface {
	List(context.Context, string) ([]string, error)
	Select(context.Context, string, string, bool) error
	Rewrite(context.Context, func(context.Context, string) (string, error), func(string, []string)) error
}
