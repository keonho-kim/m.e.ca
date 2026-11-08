// Purpose: Own bounded read-only private database handles used to restore referenced team events.
// Pattern: Page-scoped resource ownership.
// Usage: Timeline, export and tool-detail queries close their source page after hydration.
// Related: internal/storage/sqlite/team_references.go, internal/storage/sqlite/team_reference_page.go.
package sqlite

import (
	"context"
	"errors"

	"path/filepath"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (p *teamPage) close() error {
	var err error
	for _, store := range p.stores {
		err = errors.Join(err, store.Close())
	}
	return err
}
func (p *teamPage) source(ctx context.Context, id string) (*Store, error) {
	if store := p.stores[id]; store != nil {
		return store, nil
	}
	if len(p.stores) >= 8 {
		for key, store := range p.stores {
			if err := store.Close(); err != nil {
				return nil, err
			}
			delete(p.stores, key)
			break
		}
	}
	catalog, err := p.parent.teamSessions()
	if err != nil {
		return nil, err
	}
	directory, _, err := catalog.projectDirectory(p.parent.session.ProjectRoot, false)
	if err != nil {
		return nil, err
	}
	// Opening through the ordinary catalog validates IDs before constructing paths.
	if !sessionstate.ValidID(id) {
		return nil, faults.Errorf(diagcatalog.SqliteInvalidPrivateEventReference)
	}
	path := filepath.Join(directory, id+".db")
	db, err := connect(ctx, path, true)
	if err != nil {
		return nil, err
	}
	conversation, err := readConversation(ctx, db, id)
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if conversation.ProjectRoot != p.parent.session.ProjectRoot {
		return nil, errors.Join(faults.Errorf(diagcatalog.SqlitePrivateSessionProjectMismatch), db.Close())
	}
	store := &Store{db: db, path: path, session: conversation.Session, room: p.parent}
	if p.stores == nil {
		p.stores = map[string]*Store{}
	}
	p.stores[id] = store
	return store, nil
}
