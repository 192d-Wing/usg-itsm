package api_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/192d-Wing/usg-itsm/pkg/auth"
	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/api"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/domain"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/store"
	"github.com/gofiber/fiber/v2"
)

// fakeVerifier satisfies auth.Verifier, returning fixed claims.
type fakeVerifier struct{ claims *auth.Claims }

func (f fakeVerifier) Verify(_ context.Context, _ string) (*auth.Claims, error) {
	return f.claims, nil
}

// fakeStore implements api.WorkItemStore and records the last inputs.
type fakeStore struct {
	item       domain.WorkItem
	createErr  error
	transErr   error
	lastFilter store.ListFilter
	lastCreate store.CreateInput
}

func (s *fakeStore) Create(_ context.Context, in store.CreateInput) (domain.WorkItem, error) {
	s.lastCreate = in
	if s.createErr != nil {
		return domain.WorkItem{}, s.createErr
	}
	wi := s.item
	wi.Type = in.Type
	wi.RequesterID = in.RequesterID
	return wi, nil
}
func (s *fakeStore) Get(_ context.Context, _ string) (domain.WorkItem, error) {
	return s.item, nil
}
func (s *fakeStore) List(_ context.Context, f store.ListFilter) ([]domain.WorkItem, error) {
	s.lastFilter = f
	return []domain.WorkItem{s.item}, nil
}
func (s *fakeStore) Update(_ context.Context, _, _ string, _ store.Patch) (domain.WorkItem, error) {
	return s.item, nil
}
func (s *fakeStore) Transition(_ context.Context, _, _ string, to domain.Status) (domain.WorkItem, error) {
	if s.transErr != nil {
		return domain.WorkItem{}, s.transErr
	}
	wi := s.item
	wi.Status = to
	return wi, nil
}
func (s *fakeStore) AddComment(_ context.Context, _, _, _ string, _ bool) (domain.Comment, error) {
	return domain.Comment{ID: "c1"}, nil
}
func (s *fakeStore) ListComments(_ context.Context, _ string, _ bool) ([]domain.Comment, error) {
	return nil, nil
}
func (s *fakeStore) ListEvents(_ context.Context, _ string) ([]domain.Event, error) {
	return nil, nil
}

func newApp(claims *auth.Claims, st api.WorkItemStore) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.DefaultErrorHandler})
	g := app.Group("/api/v1", auth.RequireAuth(fakeVerifier{claims}))
	api.New(st).Register(g)
	return app
}

func do(t *testing.T, app *fiber.App, method, path, body string) int {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set(fiber.HeaderAuthorization, "Bearer x")
	if body != "" {
		r.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	resp, err := app.Test(r)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp.StatusCode
}

func agent() *auth.Claims {
	return &auth.Claims{Subject: "agent-1", Roles: []string{"agent"}}
}
func requester() *auth.Claims {
	return &auth.Claims{Subject: "user-1", Roles: []string{"requester"}}
}

func TestCreate_RequiresAuth(t *testing.T) {
	app := newApp(agent(), &fakeStore{})
	r := httptest.NewRequest(fiber.MethodPost, "/api/v1/tickets", strings.NewReader(`{}`))
	// no Authorization header
	resp, _ := app.Test(r)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}

func TestCreate_Valid(t *testing.T) {
	fs := &fakeStore{item: domain.WorkItem{ID: "1", Number: "INC0001001"}}
	app := newApp(requester(), fs)
	body := `{"type":"incident","title":"printer down","priority":"high"}`
	if got := do(t, app, fiber.MethodPost, "/api/v1/tickets", body); got != fiber.StatusCreated {
		t.Fatalf("want 201, got %d", got)
	}
	if fs.lastCreate.RequesterID != "user-1" {
		t.Fatalf("requester not taken from claims: %q", fs.lastCreate.RequesterID)
	}
}

func TestCreate_InvalidType(t *testing.T) {
	app := newApp(requester(), &fakeStore{})
	body := `{"type":"bogus","title":"x","priority":"high"}`
	if got := do(t, app, fiber.MethodPost, "/api/v1/tickets", body); got != fiber.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", got)
	}
}

func TestCreate_InvalidPriority(t *testing.T) {
	app := newApp(requester(), &fakeStore{})
	body := `{"type":"incident","title":"x","priority":"urgent"}`
	if got := do(t, app, fiber.MethodPost, "/api/v1/tickets", body); got != fiber.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", got)
	}
}

func TestList_RequesterScopedToOwn(t *testing.T) {
	fs := &fakeStore{item: domain.WorkItem{ID: "1", RequesterID: "user-1"}}
	app := newApp(requester(), fs)
	if got := do(t, app, fiber.MethodGet, "/api/v1/tickets", ""); got != fiber.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
	if fs.lastFilter.RequesterID != "user-1" {
		t.Fatalf("requester filter not forced: %q", fs.lastFilter.RequesterID)
	}
}

func TestGet_RequesterCannotSeeOthers(t *testing.T) {
	// Item belongs to someone else; requester must get 404.
	fs := &fakeStore{item: domain.WorkItem{ID: "1", RequesterID: "other"}}
	app := newApp(requester(), fs)
	if got := do(t, app, fiber.MethodGet, "/api/v1/tickets/11111111-1111-1111-1111-111111111111", ""); got != fiber.StatusNotFound {
		t.Fatalf("want 404, got %d", got)
	}
}

func TestTransition_RequesterForbidden(t *testing.T) {
	app := newApp(requester(), &fakeStore{})
	body := `{"status":"in_progress"}`
	if got := do(t, app, fiber.MethodPost, "/api/v1/tickets/11111111-1111-1111-1111-111111111111/transition", body); got != fiber.StatusForbidden {
		t.Fatalf("want 403, got %d", got)
	}
}

func TestTransition_InvalidIsConflict(t *testing.T) {
	fs := &fakeStore{transErr: store.ErrInvalidTransition}
	app := newApp(agent(), fs)
	body := `{"status":"closed"}`
	if got := do(t, app, fiber.MethodPost, "/api/v1/tickets/11111111-1111-1111-1111-111111111111/transition", body); got != fiber.StatusConflict {
		t.Fatalf("want 409, got %d", got)
	}
}

func TestGet_InvalidIDIsNotFound(t *testing.T) {
	app := newApp(agent(), &fakeStore{})
	if got := do(t, app, fiber.MethodGet, "/api/v1/tickets/not-a-uuid", ""); got != fiber.StatusNotFound {
		t.Fatalf("want 404 for malformed id, got %d", got)
	}
}

func TestComment_InternalRequiresAgent(t *testing.T) {
	fs := &fakeStore{item: domain.WorkItem{ID: "1", RequesterID: "user-1"}}
	app := newApp(requester(), fs)
	body := `{"body":"secret","internal":true}`
	if got := do(t, app, fiber.MethodPost, "/api/v1/tickets/11111111-1111-1111-1111-111111111111/comments", body); got != fiber.StatusForbidden {
		t.Fatalf("want 403, got %d", got)
	}
}
