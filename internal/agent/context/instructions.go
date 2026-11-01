// Purpose: Reuse the fixed system projection until its source, root or redaction changes.
// Pattern: Request-owned bounded immutable snapshot.
// Usage: Build evaluates the whole request under Projection's redaction revision barrier.
// Related: internal/agent/context/request.go, internal/agent/context/projection.go.
package context

import "strconv"

type instructionSnapshot struct{ source, root, text string }

func (p *Projection) instructions(input Input) string {
	cached := p.system
	if cached.text != "" && cached.source == input.System && cached.root == input.ProjectRoot {
		return cached.text
	}
	system := input.Redact(input.System)
	if input.ProjectRoot != "" {
		system += "\n\nProject directory (project_root): " + strconv.Quote(input.Redact(input.ProjectRoot))
	}
	p.system = instructionSnapshot{}
	// Oversized editable instructions remain accepted without another retained cache.
	if len(system) <= 256*1024 {
		p.system = instructionSnapshot{source: input.System, root: input.ProjectRoot, text: system}
	}
	return system
}
