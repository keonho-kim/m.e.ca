// Purpose: Persist model selections validated by the model-selection domain.
// Pattern: SQLite adapter.
// Usage: Connection setup and session restoration save and load selections.
// Related: internal/provider/selection/validate.go, internal/storage/sqlite/store.go.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	selectioncheck "github.com/keonho-kim/v-code/internal/provider/selection"
)

func (s *Store) SaveSelection(ctx context.Context, session string, selection modelconn.Selection) error {
	if err := selectioncheck.Validate(selection); err != nil {
		return err
	}
	body, err := json.Marshal(selection)
	if err != nil {
		return err
	}
	if len(body) > 16*1024*1024 {
		return fmt.Errorf("model selection exceeds limit")
	}
	result, err := s.db.ExecContext(ctx, "UPDATE sessions SET selection=? WHERE id=?", body, session)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("model selection session no longer exists")
	}
	return nil
}
func (s *Store) Selection(ctx context.Context, session string) (modelconn.Selection, error) {
	var body []byte
	err := s.db.QueryRowContext(ctx, "SELECT selection FROM sessions WHERE id=?", session).Scan(&body)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return modelconn.Selection{}, err
	}
	if len(body) == 0 {
		return modelconn.Selection{}, nil
	}
	if len(body) > 16*1024*1024 {
		return modelconn.Selection{}, fmt.Errorf("model selection exceeds limit")
	}
	var selected modelconn.Selection
	err = json.Unmarshal(body, &selected)
	if err != nil {
		return selected, err
	}
	return selected, selectioncheck.Validate(selected)
}
