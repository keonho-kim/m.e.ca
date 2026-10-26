// Purpose: Encode deterministic JSON HTTP bodies without unnecessary HTML escaping.
// Pattern: Protocol-local serialization configuration.
// Usage: Protocol request and native call-history encoders share these immutable options.
// Related: internal/provider/protocol/request.go, internal/provider/protocol/tools.go.
package protocol

import "github.com/bytedance/sonic"

// Requests are JSON HTTP bodies, so HTML escaping adds no protection. Sorted
// maps keep request bytes deterministic. Decoding and tool validation retain
// their own contracts rather than inheriting this encoder's configuration.
var wireJSON = sonic.Config{
	SortMapKeys:      true,
	CompactMarshaler: true,
	ValidateString:   true,
}.Froze()
