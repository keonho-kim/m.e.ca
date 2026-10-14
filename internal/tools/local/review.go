// Purpose: Build review metadata from exact frozen file contents, not rendered preview size.
// Pattern: Filesystem preparation adapter.
// Usage: Prepare calls freezeCandidate for content-changing operations.
// Related: internal/tools/local/prepare.go, internal/tools/local/diff/count.go, internal/agent/tool/candidate.go.
package local

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/tools/local/diff"
	"unicode/utf8"
)

func (op *operation) freezeCandidate(call toolapi.Call, before, after []byte) error {
	if !utf8.Valid(before) || !utf8.Valid(after) || bytes.IndexByte(before, 0) >= 0 || bytes.IndexByte(after, 0) >= 0 {
		return fmt.Errorf("Conclave requires an inspectable text change; binary changes are not supported")
	}
	added, deleted, err := diff.Count(before, after)
	if err != nil {
		return err
	}
	change := toolapi.FileDelta{Path: op.source.absolute, BeforeDigest: fmt.Sprintf("%x", sha256.Sum256(before)), AfterDigest: fmt.Sprintf("%x", sha256.Sum256(after)), Added: added, Deleted: deleted, Diff: op.approval.Preview}
	candidate := toolapi.BindCandidate(call, []toolapi.FileDelta{change})
	op.approval.Candidate = &candidate
	return nil
}
