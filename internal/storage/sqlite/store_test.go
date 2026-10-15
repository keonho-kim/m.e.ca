// Purpose: Verify private SQLite session, history and Plan recovery without models.
// Pattern: Integration tests using temporary storage.
// Usage: go test ./internal/storage/sqlite.
// Related: internal/storage/sqlite/store.go, internal/storage/sqlite/session.go, internal/storage/sqlite/plans.go.

package sqlite

import (
	"context"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionHistoryAndPlanSurviveRestart(t *testing.T) {
	ctx := context.Background()
	catalog, err := OpenSessions(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	store, err := catalog.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	session := store.Session()
	path := store.path
	if err := store.AppendMessage(ctx, session.ID, modelapi.Message{Role: "user", Content: "Fix auth"}); err != nil {
		t.Fatal(err)
	}
	plan, err := planning.NewPlan("plan_1", session.ID, "Fix auth", []string{"Read", "Change"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SavePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	revised, err := planning.RevisePlan(plan, "Fix auth safely", []string{"Read", "Test", "Change"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SavePlan(ctx, revised); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSettings(ctx, policy.Settings{Scope: policy.System}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = catalog.Open(ctx, project, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	restored := store.Session()
	if err != nil || restored.ID != session.ID {
		t.Fatalf("session: %+v, %v", restored, err)
	}
	history, err := store.Messages(ctx, session.ID, 100)
	if err != nil || len(history) != 1 || history[0].Content != "Fix auth" {
		t.Fatalf("history: %+v, %v", history, err)
	}
	recovered, err := store.LatestPlan(ctx, session.ID)
	if err != nil || recovered == nil || recovered.Revision != 2 || recovered.Status != planning.AwaitingApproval {
		t.Fatalf("plan: %+v, %v", recovered, err)
	}
	previous, err := store.PlanRevision(ctx, session.ID, "plan_1", 1)
	if err != nil || previous.Objective != "Fix auth" {
		t.Fatalf("prior revision: %+v, %v", previous, err)
	}
	settings, err := store.Settings(ctx)
	if err != nil || settings.Scope != policy.System || settings.Permission != "" {
		t.Fatalf("settings: %+v, %v", settings, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("database permissions: %v, %v", info, err)
	}
	other, err := catalog.Create(ctx, t.TempDir(), "other")
	if err != nil {
		t.Fatal(err)
	}
	if other.Session().ID == session.ID {
		t.Fatal("different projects shared one session")
	}
	if err := other.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryIsBoundedAndSessionOverridesAreValidated(t *testing.T) {
	ctx := context.Background()
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	session := store.Session()
	for _, text := range []string{"one", "two", "three"} {
		if err := store.AppendMessage(ctx, session.ID, modelapi.Message{Role: "user", Content: text}); err != nil {
			t.Fatal(err)
		}
	}
	history, err := store.Messages(ctx, session.ID, 2)
	if err != nil || len(history) != 2 || history[0].Content != "two" || history[1].Content != "three" {
		t.Fatalf("bounded history: %+v, %v", history, err)
	}
	if err := store.SaveSettings(ctx, policy.Settings{Permission: "invalid"}); err == nil {
		t.Fatal("stored invalid session policy")
	}
}

func TestStoredPlanRevisionCannotBeRewritten(t *testing.T) {
	ctx := context.Background()
	store, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	session := store.Session()
	plan, err := planning.NewPlan("plan_1", session.ID, "approved objective", []string{"read"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SavePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	plan.Objective = "different operation"
	if err := store.SavePlan(ctx, plan); err == nil {
		t.Fatal("rewrote an existing Plan revision")
	}
}
