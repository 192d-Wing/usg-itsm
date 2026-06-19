package store_test

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/db"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/domain"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSchema = "ticketing_test"

// newStore connects to TEST_DATABASE_URL, resets the test schema, and applies
// migrations. The test is skipped when no database is configured.
func newStore(t *testing.T) (*store.Store, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping store integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	pool, err := db.Connect(ctx, url, testSchema)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := pool.Exec(ctx, "DROP SCHEMA IF EXISTS "+testSchema+" CASCADE"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := db.Migrate(ctx, pool, testSchema, store.Migrations, store.MigrationsDir); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(pool.Close)
	return store.New(pool), pool
}

func TestStore_CreateTransitionCommentFlow(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()

	wi, err := st.Create(ctx, store.CreateInput{
		Type:        domain.TypeIncident,
		Title:       "printer down",
		Priority:    domain.PriorityHigh,
		RequesterID: "user-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if wi.Number != "INC0001001" {
		t.Fatalf("number = %q, want INC0001001", wi.Number)
	}
	if wi.Status != domain.StatusNew {
		t.Fatalf("status = %q, want new", wi.Status)
	}

	// Illegal transition is rejected.
	if _, err := st.Transition(ctx, wi.ID, "agent-1", domain.StatusClosed); !errors.Is(err, store.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}

	// Legal transition succeeds.
	got, err := st.Transition(ctx, wi.ID, "agent-1", domain.StatusInProgress)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if got.Status != domain.StatusInProgress {
		t.Fatalf("status = %q, want in_progress", got.Status)
	}

	if _, err := st.AddComment(ctx, wi.ID, "agent-1", "looking into it", true); err != nil {
		t.Fatalf("comment: %v", err)
	}

	// History records created + status_changed + commented.
	events, err := st.ListEvents(ctx, wi.ID)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("want 3 events, got %d", len(events))
	}
}

func TestStore_GetNotFound(t *testing.T) {
	st, _ := newStore(t)
	_, err := st.Get(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

type capturePublisher struct{ subjects []string }

func (c *capturePublisher) Publish(_ context.Context, subject string, _ []byte) error {
	c.subjects = append(c.subjects, subject)
	return nil
}

func TestStore_PublishesEvents(t *testing.T) {
	_, pool := newStore(t)
	cap := &capturePublisher{}
	st := store.New(pool, store.WithPublisher(cap))
	ctx := context.Background()

	wi, err := st.Create(ctx, store.CreateInput{
		Type: domain.TypeIncident, Title: "x", Priority: domain.PriorityLow, RequesterID: "u",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.Transition(ctx, wi.ID, "agent", domain.StatusInProgress); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if _, err := st.AddComment(ctx, wi.ID, "agent", "hi", true); err != nil {
		t.Fatalf("comment: %v", err)
	}
	assignee := "agent"
	if _, err := st.Update(ctx, wi.ID, "agent", store.Patch{AssigneeID: &assignee}); err != nil {
		t.Fatalf("update: %v", err)
	}

	want := []string{
		"itsm.ticket.created",
		"itsm.ticket.status_changed",
		"itsm.ticket.commented",
		"itsm.ticket.assigned",
	}
	for _, w := range want {
		if !slices.Contains(cap.subjects, w) {
			t.Errorf("missing event %q; got %v", w, cap.subjects)
		}
	}
}

func TestStore_RequesterIsolationViaList(t *testing.T) {
	st, _ := newStore(t)
	ctx := context.Background()
	for _, who := range []string{"user-1", "user-2", "user-1"} {
		if _, err := st.Create(ctx, store.CreateInput{
			Type: domain.TypeServiceRequest, Title: "t", Priority: domain.PriorityLow, RequesterID: who,
		}); err != nil {
			t.Fatalf("seed create: %v", err)
		}
	}
	mine, err := st.List(ctx, store.ListFilter{RequesterID: "user-1"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(mine) != 2 {
		t.Fatalf("want 2 items for user-1, got %d", len(mine))
	}
}
