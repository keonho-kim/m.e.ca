// Purpose: Own argument and execution failures for development guide search.
// Pattern: Central diagnostic catalog.
// Usage: Startup parses meca develop and the command adapter reports local search failures.
// Related: internal/commands/startup/parse.go, internal/commands/develop/search.go.
package catalog

const (
	DevelopUsage               Message = "usage: meca develop \"keywords\""
	DevelopSearchUnavailable   Message = "development search is unavailable at %q; run meca install (%w)"
	DevelopSearchMustBeRegular Message = "development search script must be a regular file at %q"
	DevelopSearchFailed        Message = "development search failed: %w"
)
