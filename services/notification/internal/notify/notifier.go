package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Notifier delivers a message over one channel.
type Notifier interface {
	Name() string
	Notify(ctx context.Context, m Message) error
}

// LogNotifier writes notifications to the log (default/dev channel).
type LogNotifier struct{ Logger *slog.Logger }

func (n LogNotifier) Name() string { return "log" }
func (n LogNotifier) Notify(_ context.Context, m Message) error {
	n.Logger.Info("notification", "title", m.Title, "body", m.Body)
	return nil
}

// WebhookNotifier POSTs the message as JSON to a URL.
type WebhookNotifier struct {
	URL    string
	Client *http.Client
}

// NewWebhookNotifier returns a webhook notifier with a bounded HTTP client.
func NewWebhookNotifier(url string) WebhookNotifier {
	return WebhookNotifier{URL: url, Client: &http.Client{Timeout: 10 * time.Second}}
}

func (n WebhookNotifier) Name() string { return "webhook" }
func (n WebhookNotifier) Notify(ctx context.Context, m Message) error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := n.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned %d", res.StatusCode)
	}
	return nil
}

// SendMailFunc matches net/smtp.SendMail; injected for testability.
type SendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// SMTPNotifier sends notifications as plaintext email via an SMTP relay.
type SMTPNotifier struct {
	Addr string
	From string
	To   []string
	Send SendMailFunc
}

// NewSMTPNotifier returns an SMTP notifier using net/smtp.
func NewSMTPNotifier(addr, from string, to []string) SMTPNotifier {
	return SMTPNotifier{Addr: addr, From: from, To: to, Send: smtp.SendMail}
}

func (n SMTPNotifier) Name() string { return "smtp" }
func (n SMTPNotifier) Notify(_ context.Context, m Message) error {
	return n.Send(n.Addr, nil, n.From, n.To, buildEmail(n.From, n.To, m))
}

func buildEmail(from string, to []string, m Message) []byte {
	return fmt.Appendf(nil,
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n",
		from, strings.Join(to, ", "), m.Title, m.Body)
}

// Dispatcher renders an event and delivers it over all configured channels.
type Dispatcher struct {
	notifiers []Notifier
	logger    *slog.Logger
}

// NewDispatcher returns a dispatcher over the given notifiers.
func NewDispatcher(logger *slog.Logger, notifiers ...Notifier) *Dispatcher {
	return &Dispatcher{notifiers: notifiers, logger: logger}
}

// Handle decodes an event and delivers it. A malformed payload is dropped
// (returns nil, so it is not redelivered); a channel failure returns an error
// so JetStream redelivers (up to the consumer's MaxDeliver). Note: redelivery
// can duplicate notifications on channels that already succeeded.
func (d *Dispatcher) Handle(subject string, data []byte) error {
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		d.logger.Warn("dropping malformed event", "subject", subject, "err", err)
		return nil
	}
	m := Render(e)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var firstErr error
	for _, n := range d.notifiers {
		if err := n.Notify(ctx, m); err != nil {
			d.logger.Error("notify failed", "channel", n.Name(), "subject", subject, "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
