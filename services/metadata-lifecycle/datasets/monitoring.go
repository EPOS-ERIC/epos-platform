package datasets

import (
	"context"
	"errors"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *Service) MonitoringDistributions(ctx context.Context, actor string) ([]domain.MonitoringConcept, error) {
	allowed, err := s.authorizer.AuthorizeGlobal(ctx, actor, domain.GlobalPermissionMonitoringRead)
	if err != nil {
		return nil, operationError("authorize monitoring", err)
	}

	if !allowed.Allowed {
		return nil, apperror.Default(apperror.CodePermissionDenied)
	}

	rows, err := s.store.MonitoringDistributions(ctx)
	if err != nil {
		return nil, operationError("load monitoring distributions", err)
	}

	concepts := make([]domain.MonitoringConcept, 0)
	conceptIndices := make(map[uuid.UUID]int)
	groupIndices := make(map[uuid.UUID]int)
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		request, err := row.Distribution.MonitoringRequest()
		if err != nil {
			id := row.Distribution.ID.String()
			s.log.ErrorContext(ctx, "failed to prepare monitoring distribution", "distribution_id", id)
			trace.SpanFromContext(ctx).RecordError(
				errors.New("failed to prepare monitoring distribution"),
				trace.WithAttributes(attribute.String("distribution_id", id)),
			)
			continue
		}

		conceptIndex, ok := conceptIndices[row.ConceptID]
		if !ok {
			conceptIndex = len(concepts)
			conceptIndices[row.ConceptID] = conceptIndex
			concepts = append(concepts, domain.MonitoringConcept{
				ID:     row.ConceptID,
				Name:   row.ConceptName,
				Groups: make([]domain.MonitoringGroup, 0),
			})
		}

		concept := &concepts[conceptIndex]

		groupIndex, ok := groupIndices[row.GroupID]
		if !ok {
			groupIndex = len(concept.Groups)
			groupIndices[row.GroupID] = groupIndex
			concept.Groups = append(concept.Groups, domain.MonitoringGroup{
				ID:            row.GroupID,
				Name:          row.GroupName,
				Distributions: make([]domain.MonitoringTarget, 0),
			})
		}

		contacts := make([]domain.ContactPoint, len(row.ContactPoints))
		copy(contacts, row.ContactPoints)
		concept.Groups[groupIndex].Distributions = append(concept.Groups[groupIndex].Distributions, domain.MonitoringTarget{
			ID:            row.Distribution.ID,
			ResourceID:    row.ResourceID,
			VersionID:     row.VersionID,
			Name:          row.Distribution.Title,
			Request:       request,
			ContactPoints: contacts,
		})
	}

	return concepts, nil
}
