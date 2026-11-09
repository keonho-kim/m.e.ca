// Purpose: Return stable session pages using bounded ordering references and page-only database reads.
// Pattern: Cursor boundary over catalog-owned immutable ordering snapshots.
// Usage: Server listings scan once and retain only IDs and update times between pages.
// Related: internal/storage/sqlite/conversation_scan.go, internal/commands/server/lifecycle.go.
package sqlite

import (
	"bytes"
	"container/heap"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"io"
	"path/filepath"
	"sort"
	"unsafe"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type conversationCursor struct {
	Version           int
	Project, Snapshot string
	After             int
}

func newerConversation(a, b conversationResult) bool {
	if a.updated.Equal(b.updated) {
		return a.conversation.ID > b.conversation.ID
	}
	return a.updated.After(b.updated)
}

type conversationHeap []conversationResult

func (h conversationHeap) Len() int           { return len(h) }
func (h conversationHeap) Less(i, j int) bool { return newerConversation(h[j], h[i]) }
func (h conversationHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *conversationHeap) Push(value any)    { *h = append(*h, value.(conversationResult)) }
func (h *conversationHeap) Pop() any {
	old := *h
	value := old[len(old)-1]
	old[len(old)-1] = conversationResult{}
	*h = old[:len(old)-1]
	return value
}

func (s *Sessions) ConversationPage(ctx context.Context, project, cursor string, limit int) ([]sessionstate.Conversation, string, error) {
	if limit < 1 || limit > 100 {
		return nil, "", faults.Errorf(diagcatalog.SqliteSessionPageLimitMustBeTo)
	}
	canonical, err := filepath.EvalSymlinks(project)
	if err != nil {
		return nil, "", err
	}
	key, err := sessionstate.ProjectKey(canonical)
	if err != nil {
		return nil, "", err
	}
	boundary, err := parseConversationCursor(cursor, key)
	if err != nil {
		return nil, "", err
	}
	if cursor != "" {
		return s.readConversationPage(ctx, canonical, key, boundary, limit)
	}
	values := make(conversationHeap, 0, limit+1)
	var refs []conversationReference
	scan, cancel := context.WithCancel(ctx)
	defer cancel()
	bytes := 0
	full := false
	err = s.scanConversations(scan, canonical, func(value conversationResult) {
		if full {
			return
		}
		refs = append(refs, conversationReference{id: value.conversation.ID, updated: value.updated})
		bytes += len(value.conversation.ID)
		if bytes+cap(refs)*int(unsafe.Sizeof(conversationReference{})) > conversationSnapshotBytes {
			full = true
			cancel()
			return
		}
		if len(values) < limit+1 {
			heap.Push(&values, value)
		} else if newerConversation(value, values[0]) {
			values[0] = value
			heap.Fix(&values, 0)
		}
	})
	if full {
		return nil, "", sessionstate.ErrPageLimit
	}
	if err != nil {
		return nil, "", err
	}
	sort.Slice(values, func(i, j int) bool { return newerConversation(values[i], values[j]) })
	next := ""
	if len(values) > limit {
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].updated.Equal(refs[j].updated) {
				return refs[i].id > refs[j].id
			}
			return refs[i].updated.After(refs[j].updated)
		})
		id, err := s.pages.put(key, refs)
		if err != nil {
			return nil, "", err
		}
		next = encodeConversationCursor(conversationCursor{Version: 1, Project: key, Snapshot: id, After: limit})
		values = values[:limit]
	}
	items := make([]sessionstate.Conversation, len(values))
	for i, value := range values {
		items[i] = value.conversation
	}
	return items, next, nil
}

func (s *Sessions) readConversationPage(ctx context.Context, project, key string, cursor conversationCursor, limit int) (items []sessionstate.Conversation, next string, err error) {
	refs, err := s.pages.get(key, cursor.Snapshot)
	if err != nil {
		return nil, "", err
	}
	if cursor.After < 1 || cursor.After >= len(refs) {
		return nil, "", sessionstate.ErrCursor
	}
	directory, _, err := s.projectDirectory(project, false)
	if err != nil {
		return nil, "", err
	}
	reader, err := openConversationReader(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { err = errors.Join(err, reader.Close()) }()
	position := cursor.After
	for position < len(refs) && len(items) < limit {
		ref := refs[position]
		position++
		value, err := reader.Read(ctx, conversationFile{path: filepath.Join(directory, ref.id+".db"), id: ref.id, group: key})
		if err != nil {
			return nil, "", err
		}
		if value.conversation.ID != "" && value.updated.Equal(ref.updated) {
			items = append(items, value.conversation)
		}
	}
	if position < len(refs) {
		cursor.After = position
		next = encodeConversationCursor(cursor)
	}
	return items, next, ctx.Err()
}

func encodeConversationCursor(value conversationCursor) string {
	body, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(body)
}

func parseConversationCursor(cursor, project string) (conversationCursor, error) {
	var value conversationCursor
	if cursor == "" {
		return value, nil
	}
	invalid := sessionstate.ErrCursor
	if len(cursor) > 1024 {
		return value, invalid
	}
	body, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return value, invalid
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, invalid
	}
	if decoder.Decode(new(any)) != io.EOF || value.Version != 1 || value.Project != project || len(value.Snapshot) != 26 || value.After < 1 {
		return value, invalid
	}
	return value, nil
}
