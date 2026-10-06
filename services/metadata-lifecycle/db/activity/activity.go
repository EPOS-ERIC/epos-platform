// Package activity provides PostgreSQL persistence for the activity feed.
package activity

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	*sqlc.Queries
	log *slog.Logger
}

func New(log *slog.Logger, pool *pgxpool.Pool) *Repo {
	return &Repo{Queries: sqlc.New(pool), log: log.With("component", "activity-repository")}
}

func (r *Repo) ListActivity(ctx context.Context, filter domain.ActivityFilter) ([]domain.ActivityEvent, error) {
	rows, err := r.Queries.ListActivity(ctx, sqlc.ListActivityParams{
		GroupID:  filter.GroupID,
		DateFrom: filter.DateFrom,
		DateTo:   filter.DateTo,
	})
	if err != nil {
		return nil, fmt.Errorf("query activity events: %w", err)
	}

	events := make([]domain.ActivityEvent, 0, len(rows))
	for _, row := range rows {
		event, err := toDomainActivityEvent(row)
		if err != nil {
			return nil, fmt.Errorf("map activity event: %w", err)
		}
		events = append(events, event)
	}
	return events, nil
}

func toDomainActivityEvent(row sqlc.ListActivityRow) (domain.ActivityEvent, error) {
	action := domain.ActivityAction(row.Action)
	if !action.Valid() {
		return domain.ActivityEvent{}, fmt.Errorf("unknown activity action %q", row.Action)
	}

	event := domain.ActivityEvent{
		Action:       action,
		OccurredAt:   row.OccurredAt,
		ActorID:      row.ActorID,
		GroupID:      row.GroupID,
		ResourceID:   row.ResourceID,
		VersionID:    row.VersionID,
		TargetUserID: row.TargetUserID,
		Role:         row.Role,
		Permission:   row.Permission,
		Note:         row.Note,
		Reason:       row.Reason,
	}
	if row.ResourceType != nil {
		resourceType := domain.ResourceType(*row.ResourceType)
		switch resourceType {
		case domain.Dataset, domain.SoftwareApplication, domain.SoftwareSourceCode:
			event.ResourceType = &resourceType
		default:
			return domain.ActivityEvent{}, fmt.Errorf("unknown activity resource type %q", *row.ResourceType)
		}
	}
	return event, nil
}
