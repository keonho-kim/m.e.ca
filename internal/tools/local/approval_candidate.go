// Purpose: Build approval metadata from exact frozen file contents, not rendered preview size.
// Pattern: Filesystem preparation adapter.
// Usage: Prepare calls freezeCandidate for content-changing operations.
// Related: internal/tools/local/prepare.go, internal/tools/local/diff/diff.go, internal/agent/tool/candidate.go.
package local

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/diff"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
	"unicode/utf8"
)

func (op *operation) freezeCandidate(call toolapi.Call, before, after []byte) error {
	if !utf8.Valid(before) || !utf8.Valid(after) || bytes.IndexByte(before, 0) >= 0 || bytes.IndexByte(after, 0) >= 0 {
		return fmt.Errorf("file tools require inspectable UTF-8 text; binary changes are not supported")
	}
	preview, added, deleted, err := diff.Analyze(op.source.absolute, before, after, op.original.exists, true)
	if err != nil {
		return err
	}
	op.approval.Preview = op.executor.redact(preview)
	op.approval.Diff = true
	countLines := func(body []byte) int {
		if len(body) == 0 {
			return 0
		}
		n := bytes.Count(body, []byte("\n"))
		if body[len(body)-1] != '\n' {
			n++
		}
		return n
	}
	change := toolapi.FileDelta{BeforeLineEndings: textformat.LineEndings(before), AfterLineEndings: textformat.LineEndings(after), FormatPolicy: op.input.FormatPolicy, BeforeLines: countLines(before), AfterLines: countLines(after), BeforeFinalNewline: bytes.HasSuffix(before, []byte("\n")), AfterFinalNewline: bytes.HasSuffix(after, []byte("\n")), Path: op.source.absolute, BeforeDigest: fmt.Sprintf("%x", sha256.Sum256(before)), AfterDigest: fmt.Sprintf("%x", sha256.Sum256(after)), Added: added, Deleted: deleted, Diff: op.approval.Preview}
	if len(before) > 0 && len(after) > 0 && (change.BeforeLineEndings != change.AfterLineEndings || change.BeforeFinalNewline != change.AfterFinalNewline) {
		op.approval.Preview += fmt.Sprintf("\nFile format: %s -> %s; final newline: %t -> %t\n", change.BeforeLineEndings, change.AfterLineEndings, change.BeforeFinalNewline, change.AfterFinalNewline)
	}
	candidate := toolapi.BindCandidate(call, []toolapi.FileDelta{change})
	op.approval.Candidate = &candidate
	return nil
}
