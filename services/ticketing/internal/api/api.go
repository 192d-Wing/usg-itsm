// Package api exposes the ticketing HTTP surface as Fiber handlers. Identity
// comes from the validated OIDC claims placed in context by pkg/auth; coarse
// RBAC is enforced here (requesters see only their own items; agents manage
// all).
package api

import (
	"context"
	"errors"

	"github.com/192d-Wing/usg-itsm/pkg/auth"
	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/domain"
	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/store"
	"github.com/gofiber/fiber/v2"
)

// Roles that grant agent-level access.
const (
	roleAgent = "agent"
	roleAdmin = "admin"
)

// Repeated response messages.
const (
	msgBadJSON   = "invalid JSON body"
	msgAgentOnly = "agent role required"
	msgNotFound  = "work item not found"
)

// WorkItemStore is the persistence surface the API depends on. *store.Store
// satisfies it; tests supply a fake.
type WorkItemStore interface {
	Create(ctx context.Context, in store.CreateInput) (domain.WorkItem, error)
	Get(ctx context.Context, id string) (domain.WorkItem, error)
	List(ctx context.Context, f store.ListFilter) ([]domain.WorkItem, error)
	Update(ctx context.Context, id, actorID string, p store.Patch) (domain.WorkItem, error)
	Transition(ctx context.Context, id, actorID string, to domain.Status) (domain.WorkItem, error)
	AddComment(ctx context.Context, workItemID, authorID, body string, internal bool) (domain.Comment, error)
	ListComments(ctx context.Context, workItemID string, includeInternal bool) ([]domain.Comment, error)
	ListEvents(ctx context.Context, workItemID string) ([]domain.Event, error)
}

// API wires HTTP handlers to the store.
type API struct {
	store WorkItemStore
}

// New returns an API backed by st.
func New(st WorkItemStore) *API {
	return &API{store: st}
}

// Register mounts the ticketing routes onto r (typically an authenticated
// /api/v1 group).
func (a *API) Register(r fiber.Router) {
	r.Post("/tickets", a.create)
	r.Get("/tickets", a.list)
	r.Get("/tickets/:id", a.get)
	r.Patch("/tickets/:id", a.update)
	r.Post("/tickets/:id/transition", a.transition)
	r.Get("/tickets/:id/comments", a.listComments)
	r.Post("/tickets/:id/comments", a.addComment)
	r.Get("/tickets/:id/events", a.listEvents)
}

func isAgent(c *auth.Claims) bool {
	return c != nil && (c.HasRole(roleAgent) || c.HasRole(roleAdmin))
}

// caller returns the validated claims; handlers run behind RequireAuth so this
// is never nil in practice, but we guard anyway.
func caller(c *fiber.Ctx) *auth.Claims {
	return auth.From(c)
}

type createReq struct {
	Type            string  `json:"type"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Priority        string  `json:"priority"`
	AssignmentGroup *string `json:"assignment_group"`
}

func (a *API) create(c *fiber.Ctx) error {
	var req createReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, "bad_request", msgBadJSON)
	}
	wiType := domain.Type(req.Type)
	if !wiType.Valid() {
		return httpx.WriteError(c, fiber.StatusUnprocessableEntity, "invalid_type",
			"type must be 'incident' or 'service_request'")
	}
	priority := domain.Priority(req.Priority)
	if !priority.Valid() {
		return httpx.WriteError(c, fiber.StatusUnprocessableEntity, "invalid_priority",
			"priority must be one of critical, high, moderate, low")
	}
	if req.Title == "" {
		return httpx.WriteError(c, fiber.StatusUnprocessableEntity, "invalid_title",
			"title is required")
	}

	wi, err := a.store.Create(c.UserContext(), store.CreateInput{
		Type:            wiType,
		Title:           req.Title,
		Description:     req.Description,
		Priority:        priority,
		RequesterID:     caller(c).Subject,
		AssignmentGroup: req.AssignmentGroup,
	})
	if err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(wi)
}

func (a *API) list(c *fiber.Ctx) error {
	f := store.ListFilter{
		Type:       domain.Type(c.Query("type")),
		Status:     domain.Status(c.Query("status")),
		AssigneeID: c.Query("assignee"),
		Limit:      c.QueryInt("limit", 50),
		Offset:     c.QueryInt("offset", 0),
	}
	// Requesters may only see their own items.
	if !isAgent(caller(c)) {
		f.RequesterID = caller(c).Subject
	} else if r := c.Query("requester"); r != "" {
		f.RequesterID = r
	}

	items, err := a.store.List(c.UserContext(), f)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (a *API) get(c *fiber.Ctx) error {
	wi, err := a.load(c)
	if err != nil {
		return err
	}
	return c.JSON(wi)
}

type patchReq struct {
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	Priority        *string `json:"priority"`
	AssigneeID      *string `json:"assignee_id"`
	AssignmentGroup *string `json:"assignment_group"`
}

func (a *API) update(c *fiber.Ctx) error {
	if !isAgent(caller(c)) {
		return httpx.WriteError(c, fiber.StatusForbidden, "forbidden", msgAgentOnly)
	}
	var req patchReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, "bad_request", msgBadJSON)
	}
	p := store.Patch{
		Title:           req.Title,
		Description:     req.Description,
		AssigneeID:      req.AssigneeID,
		AssignmentGroup: req.AssignmentGroup,
	}
	if req.Priority != nil {
		priority := domain.Priority(*req.Priority)
		if !priority.Valid() {
			return httpx.WriteError(c, fiber.StatusUnprocessableEntity, "invalid_priority",
				"priority must be one of critical, high, moderate, low")
		}
		p.Priority = &priority
	}

	wi, err := a.store.Update(c.UserContext(), c.Params("id"), caller(c).Subject, p)
	return a.respondItem(c, wi, err)
}

type transitionReq struct {
	Status  string `json:"status"`
	Comment string `json:"comment"`
}

func (a *API) transition(c *fiber.Ctx) error {
	if !isAgent(caller(c)) {
		return httpx.WriteError(c, fiber.StatusForbidden, "forbidden", msgAgentOnly)
	}
	var req transitionReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, "bad_request", msgBadJSON)
	}
	id := c.Params("id")
	wi, err := a.store.Transition(c.UserContext(), id, caller(c).Subject, domain.Status(req.Status))
	if errors.Is(err, store.ErrInvalidTransition) {
		return httpx.WriteError(c, fiber.StatusConflict, "invalid_transition", err.Error())
	}
	if req.Comment != "" && err == nil {
		if _, cerr := a.store.AddComment(c.UserContext(), id, caller(c).Subject, req.Comment, true); cerr != nil {
			return serverError(c, cerr)
		}
	}
	return a.respondItem(c, wi, err)
}

type commentReq struct {
	Body     string `json:"body"`
	Internal bool   `json:"internal"`
}

func (a *API) addComment(c *fiber.Ctx) error {
	wi, err := a.load(c)
	if err != nil {
		return err
	}
	var req commentReq
	if err := c.BodyParser(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, "bad_request", msgBadJSON)
	}
	if req.Body == "" {
		return httpx.WriteError(c, fiber.StatusUnprocessableEntity, "invalid_body", "comment body is required")
	}
	// Only agents may post internal work notes.
	if req.Internal && !isAgent(caller(c)) {
		return httpx.WriteError(c, fiber.StatusForbidden, "forbidden", "agent role required for internal notes")
	}
	cm, err := a.store.AddComment(c.UserContext(), wi.ID, caller(c).Subject, req.Body, req.Internal)
	if err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(cm)
}

func (a *API) listComments(c *fiber.Ctx) error {
	wi, err := a.load(c)
	if err != nil {
		return err
	}
	comments, err := a.store.ListComments(c.UserContext(), wi.ID, isAgent(caller(c)))
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(fiber.Map{"items": comments})
}

func (a *API) listEvents(c *fiber.Ctx) error {
	if !isAgent(caller(c)) {
		return httpx.WriteError(c, fiber.StatusForbidden, "forbidden", msgAgentOnly)
	}
	events, err := a.store.ListEvents(c.UserContext(), c.Params("id"))
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(fiber.Map{"items": events})
}

// load fetches the work item named by :id and enforces requester ownership.
func (a *API) load(c *fiber.Ctx) (domain.WorkItem, error) {
	wi, err := a.store.Get(c.UserContext(), c.Params("id"))
	if errors.Is(err, store.ErrNotFound) {
		return domain.WorkItem{}, httpx.WriteError(c, fiber.StatusNotFound, "not_found", msgNotFound)
	}
	if err != nil {
		return domain.WorkItem{}, serverError(c, err)
	}
	// Requesters may only access their own items; mask others as 404.
	if !isAgent(caller(c)) && wi.RequesterID != caller(c).Subject {
		return domain.WorkItem{}, httpx.WriteError(c, fiber.StatusNotFound, "not_found", msgNotFound)
	}
	return wi, nil
}

// respondItem maps a store result to a response, translating NotFound.
func (a *API) respondItem(c *fiber.Ctx, wi domain.WorkItem, err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return httpx.WriteError(c, fiber.StatusNotFound, "not_found", msgNotFound)
	}
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(wi)
}

func serverError(c *fiber.Ctx, err error) error {
	return httpx.WriteError(c, fiber.StatusInternalServerError, "internal_error", err.Error())
}
