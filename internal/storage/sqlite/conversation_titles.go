// Purpose: Read a bounded set of project conversation titles without scanning catalogs or loading history.
// Pattern: Scoped read-only metadata batch with shared deletion leases.
// Usage: Server task pages resolve only their distinct session identities through this lookup.
// Related: internal/storage/sqlite/conversation_reader.go, internal/storage/sqlite/delete.go.
package sqlite

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

// ConversationSummaries returns metadata for at most one bounded task page.
func (s *Sessions) ConversationSummaries(ctx context.Context, project string, ids []string) (titles map[string]string, resultErr error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(ids) > 100 {
		return nil, fault.New(diag.InvalidArgument, diag.SqliteSessionMetadataLookupLimit)
	}
	for _, id := range ids {
		if !sessionstate.ValidID(id) {
			return nil, fault.New(diag.InvalidArgument, diag.SqliteInvalidSessionID)
		}
	}
	titles = make(map[string]string, len(ids))
	if len(ids) == 0 {
		return titles, nil
	}
	directory, _, err := s.projectDirectory(project, false)
	if errors.Is(err, os.ErrNotExist) {
		return titles, nil
	}
	if err != nil {
		return nil, err
	}
	reader, err := openConversationReader(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, reader.Close()) }()
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		value, err := readCatalogTitle(ctx, reader, conversationFile{filepath.Join(directory, id+".db"), id, filepath.Base(directory)})
		if err != nil {
			return nil, err
		}
		if value.conversation.ID != "" {
			titles[id] = value.conversation.Summary
		}
	}
	return titles, ctx.Err()
}

func readCatalogTitle(ctx context.Context, reader *conversationReader, file conversationFile) (value conversationResult, resultErr error) {
	if _, err := os.Lstat(file.path); errors.Is(err, os.ErrNotExist) {
		return value, nil
	} else if err != nil {
		return value, err
	}
	lease, err := openConversationReadLock(file.path + ".guard")
	if err != nil {
		return value, err
	}
	defer func() { resultErr = errors.Join(resultErr, lease.Close()) }()
	if _, err := os.Lstat(filepath.Join(filepath.Dir(file.path), ".deleted", file.id)); err == nil {
		return value, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return value, err
	}
	value, err = reader.Read(ctx, file)
	if err != nil {
		// Failed reads can leave an attachment; close it before releasing the deletion lease.
		return value, errors.Join(err, reader.Close())
	}
	return value, nil
}
