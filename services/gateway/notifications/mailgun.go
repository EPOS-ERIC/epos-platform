// Package notifications sends notifications through Mailgun.
package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/mailgun/mailgun-go/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("github.com/EPOS-ERIC/epos-platform/services/gateway/notifications")

type Mailgun struct {
	log     *slog.Logger
	domain  string
	from    string
	client  mailgun.Mailgun
	timeout time.Duration
}

func NewMailgun(log *slog.Logger, cfg config.Mailgun) (*Mailgun, error) {
	if !cfg.Enable {
		return nil, nil
	}
	if cfg.Domain == "" || cfg.From == "" || cfg.APIKeyEnv == "" {
		return nil, fmt.Errorf("Mailgun credentials are required when enabled")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("invalid Mailgun timeout")
	}
	apiKey, ok := os.LookupEnv(cfg.APIKeyEnv)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("Mailgun API key environment variable %q is empty", cfg.APIKeyEnv)
	}

	client := mailgun.NewMailgun(apiKey)
	if err := client.SetAPIBase(mailgun.APIBaseEU); err != nil {
		return nil, fmt.Errorf("set Mailgun API base: %w", err)
	}

	client.SetHTTPClient(&http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	})

	mailer := &Mailgun{
		log:     log.With("component", "mailgun-notifications"),
		domain:  cfg.Domain,
		from:    cfg.From,
		client:  client,
		timeout: cfg.Timeout,
	}

	return mailer, nil
}

// Dispatch fires off async sends for each email notification.
func (m *Mailgun) Dispatch(ctx context.Context, notifications []domain.Notification) {
	for _, notification := range notifications {
		if notification.Channel != domain.NotificationChannelEmail {
			continue
		}
		go m.sendAsync(ctx, notification)
	}
}

func (m *Mailgun) sendAsync(ctx context.Context, notification domain.Notification) {
	ctx = context.WithoutCancel(ctx)
	ctx, span := tracer.Start(ctx, "notification.send")
	defer span.End()

	span.SetAttributes(
		attribute.String("notification.channel", string(notification.Channel)),
		attribute.String("notification.event_kind", notification.EventKind),
		attribute.String("notification.provider", "mailgun"),
		attribute.String("notification.recipient", notification.TargetEmail),
	)

	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	if err := m.send(ctx, notification); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		m.log.ErrorContext(ctx, "send notification failed", "target", notification.TargetEmail, "event", notification.EventKind, "error", err)
	}
}

func (m *Mailgun) send(ctx context.Context, notification domain.Notification) error {
	message := mailgun.NewMessage(
		m.domain,
		m.from,
		notification.Subject,
		"",
		notification.TargetEmail,
	)

	message.SetHTML(notification.Body)

	_, err := m.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("send Mailgun notification: %w", err)
	}

	return nil
}
