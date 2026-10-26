// Purpose: Share immutable Markdown grammar configuration across independent documents.
// Pattern: Shared read-only parser configuration with per-call parsing contexts.
// Usage: Full and streamed rendering parse with a fresh document/context on each call.
// Related: internal/tui/markdown/render.go, internal/tui/markdown/stream.go.
package markdown

import (
	"github.com/yuin/goldmark/v2/extension"
	"github.com/yuin/goldmark/v2/parser"
)

// Goldmark initializes grammar tables with sync.Once. Parse creates its own
// reader, context and AST; this configured parser never retains document text.
var documentParser = parser.New(parser.WithExtensions(extension.TableParser, extension.TaskListItemParser, extension.LinkifyParser))
