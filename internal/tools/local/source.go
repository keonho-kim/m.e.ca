// Purpose: Read a bounded modification source and verify it matches the prepared file version.
// Pattern: Filesystem version guard.
// Usage: File mutation preparation reads the exact bytes used for review and replacement.
// Related: internal/tools/local/prepare.go, internal/tools/local/paths.go, internal/tools/local/review.go.
package local

import (
	"crypto/sha256"
	"errors"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	fileops "github.com/keonho-kim/v-code/internal/tools/local/file"
	"io"
)

func (op *operation) readSource() (content []byte, err error) {
	file, err := op.source.root.Open(op.source.relative)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	content, err = io.ReadAll(io.LimitReader(file, fileops.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > fileops.MaxBytes || sha256.Sum256(content) != op.original.digest {
		return nil, &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "file changed while preparing its exact review contents", Next: "Read current lines again and prepare a fresh change."}
	}
	return content, nil
}
