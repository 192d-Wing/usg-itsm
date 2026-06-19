// Package store is the PostgreSQL persistence layer for ticketing. It owns the
// work-item, comment, and event tables and enforces the lifecycle state
// machine atomically inside transactions.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/192d-Wing/usg-itsm/services/ticketing/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors callers can match with errors.Is.
var (
	ErrNotFound          = errors.New("work item not found")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// Publisher emits ticket events to the message bus. The DB events table is the
// durable record; bus delivery is best-effort (ADR-0004).
type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, string, []byte) error { return nil }

// Store persists work items.
type Store struct {
	pool *pgxpool.Pool
	pub  Publisher
}

// Option configures a Store.
type Option func(*Store)

// WithPublisher sets the event publisher used to emit ticket events.
func WithPublisher(p Publisher) Option {
	return func(s *Store) { s.pub = p }
}

// New returns a Store backed by pool. By default events are persisted to the
// database only; pass WithPublisher to also emit them to the bus.
func New(pool *pgxpool.Pool, opts ...Option) *Store {
	s := &Store{pool: pool, pub: nopPublisher{}}
	for _, o := range opts {
		o(s)
	}
	return s
}

const subjectPrefix = "itsm.ticket."

// ticketEvent is the bus payload for a ticket change.
type ticketEvent struct {
	Type       string         `json:"type"`
	WorkItemID string         `json:"work_item_id"`
	Number     string         `json:"number,omitempty"`
	ActorID    string         `json:"actor_id"`
	Data       map[string]any `json:"data,omitempty"`
}

// publish emits a ticket event best-effort: a bus failure never fails the
// request, since the events table already holds the durable record.
func (s *Store) publish(ctx context.Context, kind, workItemID, number, actor string, data map[string]any) {
	b, err := json.Marshal(ticketEvent{
		Type:       "ticket." + kind,
		WorkItemID: workItemID,
		Number:     number,
		ActorID:    actor,
		Data:       data,
	})
	if err != nil {
		return
	}
	_ = s.pub.Publish(ctx, subjectPrefix+kind, b)
}

const workItemColumns = `id, number, type, title, description, status, priority,
	requester_id, assignee_id, assignment_group, created_at, updated_at, closed_at`

// CreateInput is the data needed to open a work item.
type CreateInput struct {
	Type            domain.Type
	Title           string
	Description     string
	Priority        domain.Priority
	RequesterID     string
	AssignmentGroup *string
}

// Create opens a new work item, allocating its human number from the per-type
// sequence and recording a "created" event, all in one transaction.
func (s *Store) Create(ctx context.Context, in CreateInput) (domain.WorkItem, error) {
	seqName := "incident_seq"
	if in.Type == domain.TypeServiceRequest {
		seqName = "request_seq"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.WorkItem{}, err
	}
	defer tx.Rollback(ctx)

	var seq int64
	// seqName is an internal constant, never user input — safe to interpolate.
	if err := tx.QueryRow(ctx, "SELECT nextval('"+seqName+"')").Scan(&seq); err != nil {
		return domain.WorkItem{}, fmt.Errorf("allocate number: %w", err)
	}
	number := domain.FormatNumber(in.Type, seq)

	row := tx.QueryRow(ctx, `
		INSERT INTO work_items (number, type, title, description, status, priority, requester_id, assignment_group)
		VALUES ($1, $2, $3, $4, 'new', $5, $6, $7)
		RETURNING `+workItemColumns,
		number, in.Type, in.Title, in.Description, in.Priority, in.RequesterID, in.AssignmentGroup)

	wi, err := scanWorkItem(row)
	if err != nil {
		return domain.WorkItem{}, err
	}
	if err := insertEvent(ctx, tx, wi.ID, in.RequesterID, domain.EventCreated, map[string]any{
		"type": in.Type, "priority": in.Priority,
	}); err != nil {
		return domain.WorkItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.WorkItem{}, err
	}
	s.publish(ctx, domain.EventCreated, wi.ID, wi.Number, in.RequesterID, map[string]any{
		"type": in.Type, "priority": in.Priority,
	})
	return wi, nil
}

// Get returns one work item by id, or ErrNotFound.
func (s *Store) Get(ctx context.Context, id string) (domain.WorkItem, error) {
	row := s.pool.QueryRow(ctx, "SELECT "+workItemColumns+" FROM work_items WHERE id = $1", id)
	wi, err := scanWorkItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WorkItem{}, ErrNotFound
	}
	return wi, err
}

// ListFilter narrows a work-item listing. Zero-value fields are ignored.
type ListFilter struct {
	Type        domain.Type
	Status      domain.Status
	AssigneeID  string
	RequesterID string
	Limit       int
	Offset      int
}

// List returns work items matching filter, newest first.
func (s *Store) List(ctx context.Context, f ListFilter) ([]domain.WorkItem, error) {
	var conds []string
	var args []any
	add := func(col string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if f.Type != "" {
		add("type", f.Type)
	}
	if f.Status != "" {
		add("status", f.Status)
	}
	if f.AssigneeID != "" {
		add("assignee_id", f.AssigneeID)
	}
	if f.RequesterID != "" {
		add("requester_id", f.RequesterID)
	}

	q := "SELECT " + workItemColumns + " FROM work_items"
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY created_at DESC"

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit)
	q += fmt.Sprintf(" LIMIT $%d", len(args))
	args = append(args, max(f.Offset, 0))
	q += fmt.Sprintf(" OFFSET $%d", len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.WorkItem, 0, limit)
	for rows.Next() {
		wi, err := scanWorkItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, wi)
	}
	return items, rows.Err()
}

// Patch holds optional work-item field updates; nil fields are left unchanged.
type Patch struct {
	Title           *string
	Description     *string
	Priority        *domain.Priority
	AssigneeID      *string
	AssignmentGroup *string
}

// Update applies patch and records an event. Returns the updated item.
func (s *Store) Update(ctx context.Context, id, actorID string, p Patch) (domain.WorkItem, error) {
	var sets []string
	var args []any
	set := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if p.Title != nil {
		set("title", *p.Title)
	}
	if p.Description != nil {
		set("description", *p.Description)
	}
	if p.Priority != nil {
		set("priority", *p.Priority)
	}
	if p.AssigneeID != nil {
		set("assignee_id", *p.AssigneeID)
	}
	if p.AssignmentGroup != nil {
		set("assignment_group", *p.AssignmentGroup)
	}
	if len(sets) == 0 {
		return s.Get(ctx, id)
	}
	sets = append(sets, "updated_at = now()")

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.WorkItem{}, err
	}
	defer tx.Rollback(ctx)

	args = append(args, id)
	q := "UPDATE work_items SET " + strings.Join(sets, ", ") +
		fmt.Sprintf(" WHERE id = $%d RETURNING ", len(args)) + workItemColumns
	wi, err := scanWorkItem(tx.QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WorkItem{}, ErrNotFound
	}
	if err != nil {
		return domain.WorkItem{}, err
	}

	kind := domain.EventUpdated
	if p.AssigneeID != nil {
		kind = domain.EventAssigned
	}
	if err := insertEvent(ctx, tx, id, actorID, kind, nil); err != nil {
		return domain.WorkItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.WorkItem{}, err
	}
	s.publish(ctx, kind, wi.ID, wi.Number, actorID, nil)
	return wi, nil
}

// Transition moves a work item to status to, validating the state machine and
// recording an event atomically. Returns ErrInvalidTransition on an illegal
// move and ErrNotFound when the item is absent.
func (s *Store) Transition(ctx context.Context, id, actorID string, to domain.Status) (domain.WorkItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.WorkItem{}, err
	}
	defer tx.Rollback(ctx)

	var current domain.Status
	err = tx.QueryRow(ctx, "SELECT status FROM work_items WHERE id = $1 FOR UPDATE", id).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WorkItem{}, ErrNotFound
	}
	if err != nil {
		return domain.WorkItem{}, err
	}
	if !domain.CanTransition(current, to) {
		return domain.WorkItem{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current, to)
	}

	closedClause := ""
	if to.IsTerminal() {
		closedClause = ", closed_at = now()"
	}
	q := "UPDATE work_items SET status = $1, updated_at = now()" + closedClause +
		" WHERE id = $2 RETURNING " + workItemColumns
	wi, err := scanWorkItem(tx.QueryRow(ctx, q, to, id))
	if err != nil {
		return domain.WorkItem{}, err
	}
	if err := insertEvent(ctx, tx, id, actorID, domain.EventStatusChanged, map[string]any{
		"from": current, "to": to,
	}); err != nil {
		return domain.WorkItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.WorkItem{}, err
	}
	s.publish(ctx, domain.EventStatusChanged, wi.ID, wi.Number, actorID, map[string]any{
		"from": current, "to": to,
	})
	return wi, nil
}

// AddComment appends a comment and records a "commented" event.
func (s *Store) AddComment(ctx context.Context, workItemID, authorID, body string, internal bool) (domain.Comment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Comment{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM work_items WHERE id = $1)", workItemID).Scan(&exists); err != nil {
		return domain.Comment{}, err
	}
	if !exists {
		return domain.Comment{}, ErrNotFound
	}

	var c domain.Comment
	err = tx.QueryRow(ctx, `
		INSERT INTO comments (work_item_id, author_id, body, internal)
		VALUES ($1, $2, $3, $4)
		RETURNING id, work_item_id, author_id, body, internal, created_at`,
		workItemID, authorID, body, internal).
		Scan(&c.ID, &c.WorkItemID, &c.AuthorID, &c.Body, &c.Internal, &c.CreatedAt)
	if err != nil {
		return domain.Comment{}, err
	}
	if err := insertEvent(ctx, tx, workItemID, authorID, domain.EventCommented, map[string]any{
		"internal": internal,
	}); err != nil {
		return domain.Comment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Comment{}, err
	}
	s.publish(ctx, domain.EventCommented, workItemID, "", authorID, map[string]any{
		"internal": internal,
	})
	return c, nil
}

// ListComments returns a work item's comments oldest-first. When includeInternal
// is false, internal work notes are omitted.
func (s *Store) ListComments(ctx context.Context, workItemID string, includeInternal bool) ([]domain.Comment, error) {
	q := "SELECT id, work_item_id, author_id, body, internal, created_at FROM comments WHERE work_item_id = $1"
	if !includeInternal {
		q += " AND internal = false"
	}
	q += " ORDER BY created_at ASC"

	rows, err := s.pool.Query(ctx, q, workItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Comment, 0)
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.WorkItemID, &c.AuthorID, &c.Body, &c.Internal, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListEvents returns a work item's history oldest-first.
func (s *Store) ListEvents(ctx context.Context, workItemID string) ([]domain.Event, error) {
	rows, err := s.pool.Query(ctx,
		"SELECT id, work_item_id, actor_id, kind, data, created_at FROM events WHERE work_item_id = $1 ORDER BY created_at ASC",
		workItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Event, 0)
	for rows.Next() {
		var e domain.Event
		var raw []byte
		if err := rows.Scan(&e.ID, &e.WorkItemID, &e.ActorID, &e.Kind, &raw, &e.CreatedAt); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &e.Data)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanWorkItem(r rowScanner) (domain.WorkItem, error) {
	var wi domain.WorkItem
	err := r.Scan(
		&wi.ID, &wi.Number, &wi.Type, &wi.Title, &wi.Description, &wi.Status, &wi.Priority,
		&wi.RequesterID, &wi.AssigneeID, &wi.AssignmentGroup, &wi.CreatedAt, &wi.UpdatedAt, &wi.ClosedAt,
	)
	return wi, err
}

func insertEvent(ctx context.Context, tx pgx.Tx, workItemID, actorID, kind string, data map[string]any) error {
	payload := []byte("{}")
	if len(data) > 0 {
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		payload = b
	}
	_, err := tx.Exec(ctx,
		"INSERT INTO events (work_item_id, actor_id, kind, data) VALUES ($1, $2, $3, $4)",
		workItemID, actorID, kind, payload)
	return err
}
