package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	cases := []struct {
		name      string
		event     Event
		wantTitle string
		wantBody  string
	}{
		{"created", Event{Type: "ticket.created", Number: "INC0001001", ActorID: "u1"}, "INC0001001 created", "created by u1"},
		{"status", Event{Type: "ticket.status_changed", Number: "INC0001001", ActorID: "a1", Data: map[string]any{"from": "new", "to": "in_progress"}}, "→ in_progress", "new → in_progress"},
		{"assigned", Event{Type: "ticket.assigned", Number: "REQ0000042", ActorID: "a1"}, "REQ0000042 assigned", "by a1"},
		{"commented", Event{Type: "ticket.commented", Number: "INC0001001", ActorID: "a1"}, "new comment", "from a1"},
		{"unknown", Event{Type: "ticket.weird", Number: "INC1", ActorID: "a1"}, "INC1 updated", "ticket.weird"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Render(tc.event)
			if !strings.Contains(m.Title, tc.wantTitle) {
				t.Errorf("title = %q, want contains %q", m.Title, tc.wantTitle)
			}
			if !strings.Contains(m.Body, tc.wantBody) {
				t.Errorf("body = %q, want contains %q", m.Body, tc.wantBody)
			}
		})
	}
}

func TestRender_FallsBackToIDWhenNoNumber(t *testing.T) {
	m := Render(Event{Type: "ticket.created", WorkItemID: "abc-123", ActorID: "u1"})
	if !strings.Contains(m.Title, "abc-123") {
		t.Fatalf("title = %q, want work item id", m.Title)
	}
}

func TestWebhookNotifier_PostsJSON(t *testing.T) {
	var gotBody Message
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := NewWebhookNotifier(srv.URL).Notify(context.Background(), Message{Title: "t", Body: "b"})
	if err != nil {
		t.Fatalf("notify: %v", err)
	}
	if gotCT != "application/json" {
		t.Errorf("content-type = %q", gotCT)
	}
	if gotBody.Title != "t" || gotBody.Body != "b" {
		t.Errorf("body = %+v", gotBody)
	}
}

func TestWebhookNotifier_ErrorsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if err := NewWebhookNotifier(srv.URL).Notify(context.Background(), Message{}); err == nil {
		t.Fatal("want error on 500")
	}
}

func TestSMTPNotifier_BuildsAndSends(t *testing.T) {
	var gotFrom string
	var gotTo []string
	var gotMsg []byte
	n := SMTPNotifier{
		Addr: "relay:25", From: "itsm@x", To: []string{"a@x", "b@x"},
		Send: func(_ string, _ smtp.Auth, from string, to []string, msg []byte) error {
			gotFrom, gotTo, gotMsg = from, to, msg
			return nil
		},
	}
	if err := n.Notify(context.Background(), Message{Title: "Hi", Body: "Body"}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if gotFrom != "itsm@x" || len(gotTo) != 2 {
		t.Errorf("from=%q to=%v", gotFrom, gotTo)
	}
	s := string(gotMsg)
	if !strings.Contains(s, "Subject: Hi") || !strings.Contains(s, "To: a@x, b@x") || !strings.Contains(s, "Body") {
		t.Errorf("message:\n%s", s)
	}
}

func TestDispatcher_DeliversToAll(t *testing.T) {
	a, b := &fakeNotifier{}, &fakeNotifier{}
	d := NewDispatcher(discardLogger(), a, b)
	ev, _ := json.Marshal(Event{Type: "ticket.created", Number: "INC1", ActorID: "u"})
	if err := d.Handle("itsm.ticket.created", ev); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Fatalf("calls a=%d b=%d", a.calls, b.calls)
	}
}

func TestDispatcher_MalformedIsDropped(t *testing.T) {
	a := &fakeNotifier{}
	d := NewDispatcher(discardLogger(), a)
	if err := d.Handle("itsm.ticket.created", []byte("not json")); err != nil {
		t.Fatalf("want nil (drop), got %v", err)
	}
	if a.calls != 0 {
		t.Fatalf("should not notify on malformed event")
	}
}

func TestDispatcher_ReturnsErrorWhenChannelFails(t *testing.T) {
	ok := &fakeNotifier{}
	bad := &fakeNotifier{err: errors.New("boom")}
	d := NewDispatcher(discardLogger(), ok, bad)
	ev, _ := json.Marshal(Event{Type: "ticket.created", Number: "INC1"})
	if err := d.Handle("itsm.ticket.created", ev); err == nil {
		t.Fatal("want error so the message is redelivered")
	}
	if ok.calls != 1 {
		t.Fatal("other channels should still be attempted")
	}
}

func TestDispatcher_RecoversPanic(t *testing.T) {
	d := NewDispatcher(discardLogger(), panicNotifier{})
	ev, _ := json.Marshal(Event{Type: "ticket.created", Number: "INC1"})
	err := d.Handle("itsm.ticket.created", ev) // must not panic
	if err == nil {
		t.Fatal("want error so the message is redelivered after a panic")
	}
}

func TestBuildEmail_StripsHeaderInjection(t *testing.T) {
	msg := buildEmail("a@x", []string{"b@x"}, Message{Title: "hi\r\nBcc: evil@x", Body: "body"})
	s := string(msg)
	if strings.Contains(s, "Bcc:") && strings.Contains(s, "\r\nBcc:") {
		t.Fatalf("header injection not neutralized:\n%s", s)
	}
}

type fakeNotifier struct {
	calls int
	err   error
}

func (f *fakeNotifier) Name() string { return "fake" }
func (f *fakeNotifier) Notify(context.Context, Message) error {
	f.calls++
	return f.err
}

type panicNotifier struct{}

func (panicNotifier) Name() string                          { return "panic" }
func (panicNotifier) Notify(context.Context, Message) error { panic("boom") }

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
