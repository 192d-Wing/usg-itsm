// Command notification consumes ticket events (itsm.ticket.*) from NATS
// JetStream and delivers notifications over the configured channels (ADR-0004).
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/config"
	"github.com/192d-Wing/usg-itsm/pkg/events"
	"github.com/192d-Wing/usg-itsm/pkg/log"
	"github.com/192d-Wing/usg-itsm/pkg/server"
	"github.com/192d-Wing/usg-itsm/services/notification/internal/notify"
)

func main() {
	cfg := config.Load("notification", ":8446")
	logger := log.New(cfg.ServiceName, cfg.LogLevel)

	if err := run(cfg, logger); err != nil {
		logger.Error("notification exited with error", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	if cfg.NatsURL == "" {
		return errors.New("NATS_URL is required")
	}

	dispatcher := notify.NewDispatcher(logger, buildNotifiers(cfg, logger)...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	consumer, err := events.Consume(ctx, cfg.NatsURL, events.ConsumeConfig{
		Stream:   "ITSM",
		Durable:  "notification",
		Subjects: []string{"itsm.ticket.*"},
	}, dispatcher.Handle)
	if err != nil {
		return err
	}
	defer consumer.Close()
	logger.Info("consuming ticket events", "stream", "ITSM", "durable", "notification")

	app := server.NewApp("usg-itsm-notification", nil)
	return server.Run(cfg, app, logger)
}

// buildNotifiers assembles the configured delivery channels, always including a
// log channel so events are observable even with nothing else configured.
func buildNotifiers(cfg config.Config, logger *slog.Logger) []notify.Notifier {
	notifiers := []notify.Notifier{notify.LogNotifier{Logger: logger}}

	if cfg.NotifyWebhookURL != "" {
		notifiers = append(notifiers, notify.NewWebhookNotifier(cfg.NotifyWebhookURL))
		logger.Info("webhook channel enabled", "url", cfg.NotifyWebhookURL)
	}
	if cfg.SMTPAddr != "" && cfg.NotifyEmail != "" {
		to := splitAndTrim(cfg.NotifyEmail)
		notifiers = append(notifiers, notify.NewSMTPNotifier(cfg.SMTPAddr, cfg.SMTPFrom, to))
		logger.Info("email channel enabled", "smtp", cfg.SMTPAddr, "recipients", len(to))
	}
	return notifiers
}

func splitAndTrim(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
