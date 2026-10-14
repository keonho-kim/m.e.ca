// Purpose: Parse the shared line-number and short-hash literal used by read and edit tools.
// Pattern: Tool boundary value parser.
// Usage: PFC validates anchor fields and the local snapshot store resolves them.
// Related: internal/agent/pfc/pfc.go, internal/tools/local/anchors/edit.go.
package tool

import (
	"encoding/hex"
	"strconv"
	"strings"
)

type Anchor struct {
	Line int
	Hash string
}

func ParseAnchor(text string) (Anchor, error) {
	line, hash, ok := strings.Cut(strings.TrimSpace(text), "#")
	number, err := strconv.Atoi(line)
	decoded, hashErr := hex.DecodeString(hash)
	if !ok || err != nil || number < 1 || strconv.Itoa(number) != line || hashErr != nil || len(decoded) != 4 || len(hash) != 8 || strings.ToLower(hash) != hash {
		return Anchor{}, &Failure{Code: InvalidArgument, Detail: "expected a literal line#8hex anchor", Next: "Copy the exact anchor from read_file."}
	}
	return Anchor{Line: number, Hash: hash}, nil
}
