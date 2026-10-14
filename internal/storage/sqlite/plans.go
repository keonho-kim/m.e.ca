// Purpose: Preserve every Plan revision and restore the latest session Plan.
// Pattern: Repository.
// Usage: Called by planning, approval and step execution boundaries.
// Related: internal/agent/plan/plan.go, internal/storage/sqlite/store.go, internal/storage/sqlite/store_test.go.

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
)

func (store *Store) SavePlan(ctx context.Context, plan planning.Plan) error {
	if err := plan.Check(); err != nil {
		return err
	}
	content, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	transaction, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var previous []byte
	lookupErr := transaction.QueryRowContext(ctx, "SELECT body FROM plans WHERE session_id=? AND id=? AND revision=?", plan.SessionID, plan.ID, plan.Revision).Scan(&previous)
	if lookupErr == nil {
		stored, parseErr := parsePlan(previous)
		if parseErr != nil {
			return errors.Join(parseErr, transaction.Rollback())
		}
		if !sameProposal(stored, plan) || stored.ApprovedAt != nil && (plan.ApprovedAt == nil || !stored.ApprovedAt.Equal(*plan.ApprovedAt)) {
			return errors.Join(fmt.Errorf("stored Plan revision and approval are immutable"), transaction.Rollback())
		}
	} else if !errors.Is(lookupErr, sql.ErrNoRows) {
		return errors.Join(lookupErr, transaction.Rollback())
	} else if plan.Revision > 1 {
		lookupErr := transaction.QueryRowContext(ctx, "SELECT body FROM plans WHERE session_id=? AND id=? AND revision=?", plan.SessionID, plan.ID, plan.Revision-1).Scan(&previous)
		if lookupErr != nil {
			return errors.Join(lookupErr, transaction.Rollback())
		}
		stored, parseErr := parsePlan(previous)
		if parseErr != nil {
			return errors.Join(parseErr, transaction.Rollback())
		}
		if stored.ApprovedAt != nil {
			return errors.Join(fmt.Errorf("approved Plan cannot be revised"), transaction.Rollback())
		}
	}
	_, err = transaction.ExecContext(ctx, `INSERT INTO plans(id,session_id,revision,body,updated_at) VALUES(?,?,?,?,?)
ON CONFLICT(session_id,id,revision) DO UPDATE SET body=excluded.body,updated_at=excluded.updated_at`, plan.ID, plan.SessionID, plan.Revision, content, plan.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"))
	if err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	return transaction.Commit()
}

func sameProposal(previous, current planning.Plan) bool {
	if previous.Objective != current.Objective || len(previous.Steps) != len(current.Steps) || !previous.CreatedAt.Equal(current.CreatedAt) {
		return false
	}
	for index, step := range previous.Steps {
		if step.ID != current.Steps[index].ID || step.Description != current.Steps[index].Description {
			return false
		}
	}
	return true
}

func (store *Store) LatestPlan(ctx context.Context, session string) (*planning.Plan, error) {
	var content []byte
	err := store.db.QueryRowContext(ctx, "SELECT body FROM plans WHERE session_id=? ORDER BY updated_at DESC, revision DESC LIMIT 1", session).Scan(&content)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	plan, err := parsePlan(content)
	return &plan, err
}

func (store *Store) PlanRevision(ctx context.Context, session, id string, revision int) (planning.Plan, error) {
	var content []byte
	err := store.db.QueryRowContext(ctx, "SELECT body FROM plans WHERE session_id=? AND id=? AND revision=?", session, id, revision).Scan(&content)
	if err != nil {
		return planning.Plan{}, err
	}
	return parsePlan(content)
}

func parsePlan(content []byte) (planning.Plan, error) {
	if len(content) > 512*1024 {
		return planning.Plan{}, fmt.Errorf("stored Plan exceeds size limit")
	}
	var plan planning.Plan
	if err := json.Unmarshal(content, &plan); err != nil {
		return plan, fmt.Errorf("invalid stored Plan: %w", err)
	}
	return plan, plan.Check()
}
