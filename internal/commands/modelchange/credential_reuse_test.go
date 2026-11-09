// Purpose: Measure repeated identical credential saves for a single logical connection.
// Pattern: Isolated composition probe with a real encrypted store and in-memory OS keys.
// Usage: go test with the review overlay; no system keychain is contacted.
// Related: internal/commands/modelchange/commit.go, internal/storage/credentialstore/store.go.
package modelchange

import (
	"context"
	"database/sql"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/config/credentials/credentialtest"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"net/url"
	"path/filepath"
	"testing"
)

type reviewSelection struct{ selection connection.Selection }

func (s *reviewSelection) SaveSelection(_ context.Context, _ string, v connection.Selection) error {
	s.selection = v
	return nil
}
func (s *reviewSelection) Selection(context.Context, string) (connection.Selection, error) {
	return s.selection, nil
}
func TestRepeatedCredentialRows(t *testing.T) {
	state := t.TempDir()
	v := &credentials.Vault{State: state, Keys: &credentialtest.Memory{}}
	conn, err := connection.New(connection.OpenAI, "")
	if err != nil {
		t.Fatal(err)
	}
	selected := connection.Selection{Connection: conn, Model: "fixture-model"}
	store := &reviewSelection{}
	for i := 0; i < 8; i++ {
		result, err := Commit(t.Context(), store, v, "session_fixture", selected, "fixture-api-key", nil)
		if err != nil {
			t.Fatal(err)
		}
		selected = result.Selection
	}
	u := url.URL{Scheme: "file", Path: filepath.Join(state, "credentials.db"), RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM credentials").Scan(&n); err != nil {
		t.Fatal(err)
	}
	t.Logf("one logical connection, eight identical saves: %d credential rows", n)
	if n != 1 {
		t.Fatalf("identical saves allocate additional credential rows: %d", n)
	}
}
