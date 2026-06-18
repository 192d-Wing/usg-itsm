// Package domain holds the ticketing core model: work items (incidents and
// service requests share one type per ADR-0008), their lifecycle state
// machine, and pure helpers. It has no I/O so it is exhaustively unit-tested.
package domain

import (
	"fmt"
	"slices"
	"time"
)

// Type discriminates the two MVP work-item kinds.
type Type string

const (
	TypeIncident       Type = "incident"
	TypeServiceRequest Type = "service_request"
)

// Valid reports whether t is a known work-item type.
func (t Type) Valid() bool {
	return t == TypeIncident || t == TypeServiceRequest
}

// NumberPrefix returns the human-facing number prefix for the type.
func (t Type) NumberPrefix() string {
	if t == TypeServiceRequest {
		return "REQ"
	}
	return "INC"
}

// Status is a work-item lifecycle state.
type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusOnHold     Status = "on_hold"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
	StatusCancelled  Status = "cancelled"
)

// Priority is the work-item priority (P1=critical … P4=low).
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityModerate Priority = "moderate"
	PriorityLow      Priority = "low"
)

// Valid reports whether p is a known priority.
func (p Priority) Valid() bool {
	switch p {
	case PriorityCritical, PriorityHigh, PriorityModerate, PriorityLow:
		return true
	default:
		return false
	}
}

// transitions is the allowed state machine. A terminal state (closed) has no
// outgoing edges; cancelled is reachable from any non-terminal state.
var transitions = map[Status][]Status{
	StatusNew:        {StatusInProgress, StatusOnHold, StatusResolved, StatusCancelled},
	StatusInProgress: {StatusOnHold, StatusResolved, StatusCancelled},
	StatusOnHold:     {StatusInProgress, StatusResolved, StatusCancelled},
	StatusResolved:   {StatusInProgress, StatusClosed}, // reopen or close
	StatusClosed:     {},
	StatusCancelled:  {},
}

// IsTerminal reports whether s has no outgoing transitions.
func (s Status) IsTerminal() bool {
	return s == StatusClosed || s == StatusCancelled
}

// AllowedTransitions returns the states reachable from s in one step.
func AllowedTransitions(s Status) []Status {
	return transitions[s]
}

// CanTransition reports whether from → to is a legal single step.
func CanTransition(from, to Status) bool {
	return slices.Contains(transitions[from], to)
}

// FormatNumber renders a zero-padded human number, e.g. "INC0001001".
func FormatNumber(t Type, seq int64) string {
	return fmt.Sprintf("%s%07d", t.NumberPrefix(), seq)
}

// WorkItem is an incident or service request.
type WorkItem struct {
	ID              string     `json:"id"`
	Number          string     `json:"number"`
	Type            Type       `json:"type"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Status          Status     `json:"status"`
	Priority        Priority   `json:"priority"`
	RequesterID     string     `json:"requester_id"`
	AssigneeID      *string    `json:"assignee_id,omitempty"`
	AssignmentGroup *string    `json:"assignment_group,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
}

// Comment is a note on a work item. Internal comments are agent-only work
// notes; non-internal comments are visible to the requester.
type Comment struct {
	ID         string    `json:"id"`
	WorkItemID string    `json:"work_item_id"`
	AuthorID   string    `json:"author_id"`
	Body       string    `json:"body"`
	Internal   bool      `json:"internal"`
	CreatedAt  time.Time `json:"created_at"`
}

// Event is an immutable history/audit record for a work item.
type Event struct {
	ID         string         `json:"id"`
	WorkItemID string         `json:"work_item_id"`
	ActorID    string         `json:"actor_id"`
	Kind       string         `json:"kind"`
	Data       map[string]any `json:"data,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Event kinds.
const (
	EventCreated       = "created"
	EventStatusChanged = "status_changed"
	EventAssigned      = "assigned"
	EventCommented     = "commented"
	EventUpdated       = "updated"
)
