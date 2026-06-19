// Package notify renders ticket events into human messages and delivers them
// over one or more channels. It consumes the events published by the ticketing
// service (ADR-0004).
package notify

import "fmt"

// Event mirrors the ticketing bus payload (itsm.ticket.*).
type Event struct {
	Type       string         `json:"type"`
	WorkItemID string         `json:"work_item_id"`
	Number     string         `json:"number"`
	ActorID    string         `json:"actor_id"`
	Data       map[string]any `json:"data"`
}

// Message is a rendered, channel-agnostic notification.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Render turns an event into a message.
func Render(e Event) Message {
	ref := e.Number
	if ref == "" {
		ref = e.WorkItemID
	}
	switch e.Type {
	case "ticket.created":
		return Message{
			Title: fmt.Sprintf("%s created", ref),
			Body:  fmt.Sprintf("%s was created by %s.", ref, e.ActorID),
		}
	case "ticket.status_changed":
		from, to := str(e.Data["from"]), str(e.Data["to"])
		return Message{
			Title: fmt.Sprintf("%s → %s", ref, to),
			Body:  fmt.Sprintf("%s changed status %s → %s (by %s).", ref, from, to, e.ActorID),
		}
	case "ticket.assigned":
		return Message{
			Title: fmt.Sprintf("%s assigned", ref),
			Body:  fmt.Sprintf("%s assignment changed by %s.", ref, e.ActorID),
		}
	case "ticket.commented":
		return Message{
			Title: fmt.Sprintf("%s — new comment", ref),
			Body:  fmt.Sprintf("%s received a new comment from %s.", ref, e.ActorID),
		}
	default:
		return Message{
			Title: fmt.Sprintf("%s updated", ref),
			Body:  fmt.Sprintf("%s: %s (by %s).", ref, e.Type, e.ActorID),
		}
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
