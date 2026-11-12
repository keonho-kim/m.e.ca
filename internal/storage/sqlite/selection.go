// Purpose: Persist model selections validated by the model-selection domain.
// Pattern: SQLite adapter.
// Usage: Connection setup and session restoration save and load selections.
// Related: internal/provider/selection/validate.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"io"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	selectioncheck "github.com/keonho-kim/meca/internal/provider/selection"
)

func (s *Store) SaveSelection(ctx context.Context, session string, selection modelconn.Selection) error {
	if err := s.checkSession(session); err != nil {
		return err
	}
	if err := selectioncheck.Validate(selection); err != nil {
		return err
	}
	body, err := json.Marshal(selection)
	if err != nil {
		return err
	}
	if len(body) > 16*1024*1024 {
		return faults.Errorf(diagcatalog.SqliteModelSelectionExceedsLimit)
	}
	result, err := s.execWrite(ctx, "UPDATE sessions SET selection=? WHERE id=?", body, session)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return faults.Errorf(diagcatalog.SqliteModelSelectionSessionNoLongerExists)
	}
	return nil
}
func (s *Store) Selection(ctx context.Context, session string) (modelconn.Selection, error) {
	if err := s.checkSession(session); err != nil {
		return modelconn.Selection{}, err
	}
	var body []byte
	err := s.db.QueryRowContext(ctx, "SELECT selection FROM sessions WHERE id=?", session).Scan(&body)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return modelconn.Selection{}, err
	}
	if len(body) == 0 {
		return modelconn.Selection{}, nil
	}
	if len(body) > 16*1024*1024 {
		return modelconn.Selection{}, faults.Errorf(diagcatalog.SqliteModelSelectionExceedsLimit)
	}
	var selected modelconn.Selection
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&selected)
	if err != nil {
		return selected, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return selected, faults.Errorf(diagcatalog.SqliteInvalidTrailingModelSelectionContent)
	}
	return selected, selectioncheck.Validate(selected)
}
