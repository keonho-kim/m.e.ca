// Purpose: Define independent review memory and its original-source persistence boundary.
// Pattern: Consumer-owned repository and immutable source values.
// Usage: LAT memory preparation reads users and actual outcomes without the main summary.
// Related: internal/agent/lat/memory/controller.go, internal/storage/sqlite/lat_memory.go.
package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const MaxBytes = 16 * 1024 * 1024
const TargetBytes = 32 * 1024

var ErrSourceBudget = errors.New("original review record cannot fit available input; original retained")

type State struct {
	Revision int
	Through  int64
	Text     string
}

func (s State) Check() error {
	if s.Revision < 0 || s.Through < 0 || len(s.Text) > MaxBytes || !utf8.ValidString(s.Text) || s.Revision == 0 && (s.Through != 0 || s.Text != "") || s.Revision > 0 && (s.Through == 0 || strings.TrimSpace(s.Text) == "") {
		return fmt.Errorf("invalid independent review memory")
	}
	return nil
}

type Record struct {
	Sequence int64  `json:"-"`
	Kind     string `json:"kind"`
	Text     string `json:"text"`
}

type Mark struct{ Through, Bytes, LatestUser int64 }

type Repository interface {
	LATMemory(context.Context, string) (State, error)
	InstallLATMemory(context.Context, string, int, State) error
	LATWatermark(context.Context, string, int64) (Mark, error)
	LATRecords(context.Context, string, int64, int64, int) ([]Record, int64, error)
}

// Excerpt bounds observations, never the accepted user request or persisted source.
func Excerpt(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	end := max(0, limit-40)
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + "\n[excerpt; original record retained]"
}
