// Purpose: Search complete visible narratives with bounded reads and resumable original pages.
// Pattern: Redacted read projection over the existing timeline and detail cache.
// Usage: SearchHistory never indexes or duplicates conversation bodies in another database.
// Related: internal/serverapi/detail.go, webui/src/pages/workspace/ui/chat/history-search.tsx.
package serverapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

type searchPosition struct {
	Before   int64
	Sequence int64
	Offset   int64
	Thinking bool
	Revision string
	Query    string
}

func (a *API) searchHistory(ctx context.Context, request *v1.HistorySearch) (*v1.HistoryMatches, error) {
	if len(request.GetQuery()) > 256 {
		return nil, fault.New(diag.InvalidArgument, diag.ServerSearchQueryInvalid)
	}
	query := strings.TrimSpace(request.GetQuery())
	if len(query) == 0 || !utf8.ValidString(query) {
		return nil, fault.New(diag.InvalidArgument, diag.ServerSearchQueryInvalid)
	}
	if len(request.GetCursor()) > 1024 {
		return nil, fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
	}
	hash := sha256.Sum256([]byte(a.ScopeID + "/" + request.GetSessionId() + "/" + query))
	position := searchPosition{Query: hex.EncodeToString(hash[:])}
	if request.GetCursor() != "" {
		body, err := base64.RawURLEncoding.DecodeString(request.GetCursor())
		var decoded searchPosition
		if err != nil || decodeExactJSON(body, &decoded) != nil || decoded.Query != hex.EncodeToString(hash[:]) || decoded.Before <= 0 || decoded.Sequence < 0 || decoded.Offset < 0 || len(decoded.Revision) > 64 || decoded.Offset > 0 && decoded.Revision == "" || decoded.Sequence > 0 && decoded.Before != decoded.Sequence+1 || decoded.Sequence == 0 && (decoded.Offset != 0 || decoded.Revision != "") {
			return nil, fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
		}
		position = decoded
	}
	values, _, err := a.Service.History(ctx, request.GetSessionId(), position.Before, 16)
	if err != nil {
		return nil, err
	}
	result := &v1.HistoryMatches{}
	remaining := 512 << 10
	needle := strings.ToLower(query)
	for index := len(values) - 1; index >= 0; index-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value := values[index]
		item := historyItem(value)
		a.sanitize(item)
		message := item.GetMessage()
		if message == nil {
			message = item.GetEvent().GetMessage()
		}
		if message == nil {
			position.Before = value.Sequence
			continue
		}
		for _, thinking := range []bool{false, true} {
			if position.Sequence == value.Sequence && position.Thinking && !thinking {
				continue
			}
			text := message.GetText()
			if thinking {
				text = message.GetThinking()
			}
			if text == "" {
				continue
			}
			if !item.GetTruncated() {
				if excerpt, _, found := searchExcerpt(text, needle); found {
					result.Items = append(result.Items, &v1.HistoryMatch{Sequence: value.Sequence, Thinking: thinking, Excerpt: excerpt})
				}
				remaining -= len(text)
				continue
			}
			offset, revision := int64(0), ""
			if position.Sequence == value.Sequence && position.Thinking == thinking {
				offset, revision = position.Offset, position.Revision
			}
			view := v1.DetailRequest_TEXT
			if thinking {
				view = v1.DetailRequest_THINKING
			}
			for {
				page, err := a.readDetail(ctx, &v1.DetailRequest{SessionId: request.GetSessionId(), Source: &v1.DetailRequest_HistorySequence{HistorySequence: value.Sequence}, View: view, Offset: offset, Revision: revision, Limit: 32768})
				if err != nil {
					return nil, err
				}
				remaining -= len(page.Text)
				if excerpt, foundAt, found := searchExcerpt(page.Text, needle); found {
					result.Items = append(result.Items, &v1.HistoryMatch{Sequence: value.Sequence, Thinking: thinking, Excerpt: excerpt, Offset: offset + int64(foundAt), Revision: page.Revision})
					break
				}
				if page.NextOffset >= page.TotalBytes {
					break
				}
				overlap := max(0, len(page.Text)-1024)
				for overlap > 0 && !utf8.RuneStart(page.Text[overlap]) {
					overlap--
				}
				if overlap == 0 {
					return nil, fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
				}
				offset += int64(overlap)
				revision = page.Revision
				if remaining <= 0 {
					position.Before, position.Sequence, position.Offset, position.Thinking, position.Revision = value.Sequence+1, value.Sequence, offset, thinking, revision
					return searchContinuation(result, position), nil
				}
			}
		}
		position.Before, position.Sequence, position.Offset, position.Revision = value.Sequence, 0, 0, ""
		if remaining <= 0 {
			return searchContinuation(result, position), nil
		}
	}
	if len(values) == 16 {
		return searchContinuation(result, position), nil
	}
	return result, nil
}

func searchContinuation(result *v1.HistoryMatches, position searchPosition) *v1.HistoryMatches {
	body, _ := json.Marshal(position)
	result.Cursor = base64.RawURLEncoding.EncodeToString(body)
	return result
}

func searchExcerpt(text, needle string) (string, int, bool) {
	lower := strings.ToLower(text)
	at := strings.Index(lower, needle)
	if at < 0 {
		return "", 0, false
	}
	runes := []rune(text)
	index := utf8.RuneCountInString(lower[:at])
	start := max(0, index-60)
	end := min(len(runes), index+160)
	return string(runes[start:end]), len(string(runes[:start])), true
}
