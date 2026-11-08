// Purpose: Own provider progress-watch diagnostics independently of network-byte activity.
// Pattern: Declarative diagnostic catalog.
// Usage: ContentWatch reports semantic generation stalls through stable error classes.
// Related: internal/provider/transport/content_watch.go.
package catalog

const ProviderNoContentProgress Message = "provider sent no generation progress for %s"
