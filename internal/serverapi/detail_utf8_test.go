// Purpose: Verify text pages advance in source bytes without splitting UTF-8 or requiring Base64.
// Pattern: Native API pagination integration test.
// Usage: go test ./internal/serverapi -run TestDetailTextPages.
// Related: internal/serverapi/detail.go, api/meca/v1/state.proto.
package serverapi

import (
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/storage/history"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDetailTextPagesPreserveUTF8AndByteOffsets(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	state, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("한글 😀 ", 24)
	if err := f.store.Commit(t.Context(), state.Session.ID, []record.Entry{{Message: &model.Message{Role: "assistant", Content: text}}}); err != nil {
		t.Fatal(err)
	}
	items, _, err := f.store.TimelineSummaries(t.Context(), state.Session.ID, 0, 1, history.ReadBounds{})
	if err != nil || len(items) != 1 {
		t.Fatal(items, err)
	}
	offset := int64(0)
	revision := ""
	var output strings.Builder
	for {
		reply, err := f.api.Call(t.Context(), &v1.Command{Operation: &v1.Command_Detail{Detail: &v1.DetailRequest{SessionId: state.Session.ID, Source: &v1.DetailRequest_HistorySequence{HistorySequence: items[0].Sequence}, Offset: offset, Limit: 7, Revision: revision}}})
		if err != nil || reply.GetError() != nil {
			t.Fatal(reply, err)
		}
		page := reply.GetDetail()
		if !utf8.ValidString(page.Text) || page.NextOffset <= offset || page.NextOffset-offset > 7 {
			t.Fatal("invalid or stalled text page", page)
		}
		output.WriteString(page.Text)
		offset = page.NextOffset
		revision = page.Revision
		if offset >= page.TotalBytes {
			break
		}
	}
	if !json.Valid([]byte(output.String())) || !strings.Contains(output.String(), text) {
		t.Fatal("paged text did not reconstruct the original document")
	}
}
