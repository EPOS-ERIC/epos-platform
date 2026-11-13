// Package metadatasync mirrors remote metadata into metadata lifecycle.
package metadatasync

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/config"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type Client interface {
	SyncOrganizations(context.Context, []domain.SyncedOrganization) error
}

type Worker struct {
	metadata   Client
	config     config.MetadataSync
	password   string
	httpClient *http.Client
	log        *slog.Logger
	trigger    chan struct{}
	schedule   cron.Schedule
}

func NewWorker(metadata Client, cfg config.MetadataSync, log *slog.Logger) (*Worker, error) {
	schedule, err := cron.ParseStandard(cfg.Cron)
	if err != nil {
		return nil, fmt.Errorf("parse metadata sync cron: %w", err)
	}

	password := os.Getenv(cfg.PasswordEnv)
	if password == "" {
		return nil, fmt.Errorf("organization sync password environment variable %q is empty", cfg.PasswordEnv)
	}

	return &Worker{
		metadata: metadata,
		config:   cfg,
		password: password,
		log:      log.With("component", "organization-sync"),
		trigger:  make(chan struct{}, 1),
		schedule: schedule,
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}, nil
}

// Trigger schedules a sync after checking both global management permissions.
func (w *Worker) Trigger(user domain.User, authenticated bool) error {
	if !authenticated || user.ID == "" {
		return apperror.Default(apperror.CodeUnauthenticated)
	}

	for _, permission := range []domain.GlobalPermission{
		domain.GlobalPermissionOrganizationManage,
		domain.GlobalPermissionContactPointManage,
	} {
		if !slices.Contains(user.GlobalGrantedPermissions, permission) && !slices.Contains(user.GlobalRolePermissions, permission) {
			return apperror.Default(apperror.CodePermissionDenied)
		}
	}

	if w == nil {
		return apperror.New(apperror.CodeUpstreamUnavailable, "Metadata synchronization is disabled.")
	}

	select {
	case w.trigger <- struct{}{}:
	default:
	}

	return nil
}

// Run syncs on the configured cron schedule or on request until cancellation.
func (w *Worker) Run(ctx context.Context) {
	for {
		next := w.schedule.Next(time.Now().UTC())
		if next.IsZero() {
			w.log.ErrorContext(ctx, "metadata sync cron has no future execution time")
			return
		}

		timer := time.NewTimer(time.Until(next))

		select {
		case <-ctx.Done():
			timer.Stop()
			return

		case <-timer.C:
		case <-w.trigger:
			timer.Stop()
		}

		w.runOnce(ctx)
	}
}

func (w *Worker) runOnce(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, w.config.Timeout)
	defer cancel()

	ctx, span := otel.Tracer("gateway/metadatasync").Start(ctx, "metadata.sync")
	defer span.End()

	start := time.Now()

	w.log.InfoContext(ctx, "organization sync started")

	// TODO: add contact point sync too when we have the remote api
	count, pages, err := w.syncOrganizations(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "organization sync failed")

		w.log.ErrorContext(ctx, "organization sync failed", "error", err, "pages", pages, "organizations", count, "duration", time.Since(start).Milliseconds())

		return
	}

	w.log.InfoContext(ctx, "organization sync completed", "pages", pages, "organizations", count, "duration", time.Since(start).Milliseconds())
}
