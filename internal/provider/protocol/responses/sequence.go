// Purpose: Identify retransmitted Responses events while detecting unseen frames delivered out of order.
// Pattern: Attempt-owned ordered sequence ranges.
// Usage: Decoder checks numbered SSE frames before applying content or tool fragments.
// Related: internal/provider/protocol/responses/stream.go.
package responses

import (
	"sort"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type sequenceRange struct{ first, last int64 }

func (s *Decoder) seenSequence(n int64) (bool, error) {
	if n < 0 {
		return false, fault.Errorf(diag.ResponsesInvalidSequence)
	}
	if length := len(s.sequences); length > 0 {
		last := &s.sequences[length-1]
		if n <= last.last {
			i := sort.Search(length, func(i int) bool { return s.sequences[i].last >= n })
			if n < s.sequences[i].first {
				return false, fault.Errorf(diag.ResponsesInvalidSequence)
			}
			return true, nil
		}
		if n-last.last == 1 {
			last.last = n
			return false, nil
		}
	}
	// An ordered stream normally needs one range, even over millions of tokens.
	if len(s.sequences) >= 1024 {
		return false, fault.Errorf(diag.ResponsesInvalidSequence)
	}
	s.sequences = append(s.sequences, sequenceRange{n, n})
	return false, nil
}
