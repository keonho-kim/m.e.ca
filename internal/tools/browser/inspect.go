// Purpose: Read native observations and reuse unchanged snapshots without retaining delivered page contents.
// Pattern: Session-owned browser observation adapter.
// Usage: Session.Run holds the gate while the helper returns its scoped browser state.
// Related: internal/tools/browser/capture.go, internal/tools/browser/results.go.
package browser

import (
	"context"
	"encoding/json"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

type snapshotBaseline struct {
	options snapshotOptions
	address string
	result  string
	version uint64
	mask    int
}

type snapshotOptions struct {
	selector string
	depth    int
}

type nativeSnapshot struct {
	Origin   string
	Snapshot struct {
		Kind         string
		Revision     uint64
		BaseRevision uint64
		Tree         *string
	}
}

func (s *Session) inspectWithURL(ctx context.Context, limit int, args []string) (captured, error) {
	input, err := json.Marshal([2][]string{{"get", "url"}, args})
	if err != nil {
		return captured{}, err
	}
	return s.capture(ctx, input, limit, "batch", "--bail", "--omit-command")
}

func (s *Session) inspect(ctx context.Context, r Request) (Page, error) {
	switch r.View {
	case "snapshot":
		return s.inspectSnapshot(ctx, r)
	case "screenshot":
		return s.screenshot(ctx)
	}
	data, err := s.inspectWithURL(ctx, maxCapture+(32<<10), r.args())
	if err != nil {
		return Page{}, err
	}
	defer data.Close()
	return s.store(data.data, r.Limit)
}

func (s *Session) inspectSnapshot(ctx context.Context, r Request) (Page, error) {
	options := snapshotOptions{selector: r.Selector, depth: r.snapshotDepth()}
	mask := s.maskVersion()
	args := append(r.args(), "--delta")
	for _, force := range []bool{false, true} {
		if force {
			args = append(args, "--full")
		}
		data, err := s.invoke(ctx, args...)
		if err != nil {
			return Page{}, err
		}
		page, needsFull, err := s.acceptSnapshot(data.data, options, mask, r.Limit)
		data.Close()
		if !needsFull || err != nil {
			return page, err
		}
		// The helper kept its own baseline. MECA has released delivered pages
		// and needs a fresh complete tree to represent a changed observation.
	}
	return Page{}, invalidResponse()
}

func (s *Session) acceptSnapshot(body []byte, options snapshotOptions, mask, limit int) (Page, bool, error) {
	memory := processmemory.NewPayloadLease()
	defer memory.Close()
	if err := memory.Resize(4 * len(body)); err != nil {
		return Page{}, false, err
	}
	var value nativeSnapshot
	if err := json.Unmarshal(body, &value); err != nil {
		return Page{}, false, err
	}
	if value.Origin == "" || value.Snapshot.Revision == 0 {
		return Page{}, false, invalidResponse()
	}
	s.setAddress(value.Origin)
	address := s.redact(value.Origin)
	if mask != s.maskVersion() {
		return Page{}, false, fault.Errorf(diag.BrowserResult)
	}
	revision := value.Snapshot.Revision
	switch value.Snapshot.Kind {
	case "unchanged":
		baseline := s.snapshot
		if baseline.options == options && baseline.address == address && baseline.mask == mask && baseline.version == value.Snapshot.BaseRevision {
			for _, r := range s.results {
				if r.id == baseline.result {
					s.snapshot.version = revision
					return r.unchanged(), false, nil
				}
			}
		}
		return Page{}, true, nil
	case "delta":
		return Page{}, true, nil
	case "full":
		if value.Snapshot.Tree != nil {
			page, err := s.storeText(*value.Snapshot.Tree, address, mask, [32]byte{}, limit)
			if err == nil {
				s.snapshot = snapshotBaseline{options: options, address: address, result: page.ResultID, version: revision, mask: mask}
			}
			return page, false, err
		}
	}
	return Page{}, false, invalidResponse()
}
