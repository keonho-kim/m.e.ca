// Purpose: Cache selected-path ordering and bounded display rows until their inputs change.
// Pattern: Event-loop presentation cache.
// Usage: Chat and file composers share this projection; selection commits own its revision.
// Related: internal/tui/session/file_selection.go, internal/tui/composer/request.go, internal/tui/view/references.go.
package session

import (
	"maps"
	"slices"

	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/view"
)

type referenceProjection struct {
	loaded                 bool
	revision               uint64
	paths, lines           []string
	width, rows, redaction int
	language               locale.Language
}

func (a *app) selectedPaths() []string {
	p := &a.references
	if !p.loaded || p.revision != a.referenceRevision || len(p.paths) != len(a.browser.Selected) {
		*p = referenceProjection{loaded: true, revision: a.referenceRevision, paths: slices.Sorted(maps.Keys(a.browser.Selected))}
	}
	return p.paths
}

func (a *app) selectedReferenceLines(width, rows int) []string {
	paths := a.selectedPaths()
	p := &a.references
	version := a.opts.Vault.RedactionVersion()
	if p.lines != nil && p.width == width && p.rows == rows && p.language == a.language && p.redaction == version {
		return p.lines
	}
	lines := view.ReferenceLines(paths, width, rows, a.language, a.opts.Vault.Redact)
	p.width, p.rows, p.language, p.redaction = width, rows, a.language, version
	p.lines = lines
	return lines
}
