// Package metadataimport coordinates authorized, atomic metadata imports.
package metadataimport

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/authz"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/google/uuid"
)

var (
	ErrDenied   = errors.New("permission denied")
	ErrConflict = errors.New("import conflict")
	ErrInvalid  = errors.New("invalid metadata import")
	ErrInternal = errors.New("metadata import failed")
)

type Options struct {
	BaseURI string
}

type Batch struct {
	CategorySchemes      []metadata.CategoryScheme
	Categories           []metadata.Category
	Organizations        []metadata.Organization
	ContactPoints        []metadata.ContactPoint
	Datasets             []metadata.Dataset
	SoftwareApplications []metadata.SoftwareApplication
	SoftwareSourceCodes  []metadata.SoftwareSourceCode
	CategoryEdges        []CategoryEdge
	OrganizationEdges    []OrganizationEdge
	Warnings             []Diagnostic
}

type CategoryEdge struct {
	ParentID uuid.UUID
	ChildID  uuid.UUID
}

type OrganizationEdge struct {
	ParentID uuid.UUID
	ChildID  uuid.UUID
}

type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Diagnostic struct {
	Severity  Severity
	Subject   string
	Predicate string
	Message   string
}

type Result struct {
	Groups               int
	Categories           int
	Organizations        int
	ContactPoints        int
	Datasets             int
	SoftwareApplications int
	SoftwareSourceCodes  int
	Warnings             []Diagnostic
}

type Parser interface {
	Parse(io.Reader, Options) (Batch, error)
}

type Authorizer interface {
	AuthorizeAdmin(context.Context, uuid.UUID) (authz.Result, error)
}

type Repository interface {
	Import(context.Context, uuid.UUID, Batch) (Result, error)
}

type Service struct {
	parser     Parser
	authorizer Authorizer
	repository Repository
}

func New(parser Parser, authorizer Authorizer, repository Repository) *Service {
	return &Service{parser: parser, authorizer: authorizer, repository: repository}
}

func (s *Service) Import(ctx context.Context, actorID uuid.UUID, input io.Reader) (Result, error) {
	decision, err := s.authorizer.AuthorizeAdmin(ctx, actorID)
	if err != nil {
		return Result{}, operationError("authorize import", err)
	}
	if !decision.Allowed {
		return Result{}, fmt.Errorf("%w: %s", ErrDenied, decision.Reason)
	}

	batch, err := s.parser.Parse(input, Options{BaseURI: ""})
	if err != nil {
		return Result{}, fmt.Errorf("parse Turtle: %w: %w", ErrInvalid, err)
	}
	if err := validateResourceGroups(batch); err != nil {
		return Result{}, fmt.Errorf("validate ownership: %w: %w", ErrInvalid, err)
	}

	result, err := s.repository.Import(ctx, actorID, batch)
	return result, operationError("persist import", err)
}

func validateDatasetGroups(batch Batch) error {
	for _, dataset := range batch.Datasets {
		schemes := make(map[string]struct{})
		for _, category := range dataset.Categories {
			schemes[category.Scheme.SourceIRI] = struct{}{}
		}
		if len(schemes) != 1 {
			return fmt.Errorf("dataset %q must reference categories from exactly one scheme", dataset.SourceIRI)
		}
	}
	return nil
}

func validateResourceGroups(batch Batch) error {
	if err := validateDatasetGroups(batch); err != nil {
		return err
	}
	for _, software := range batch.SoftwareApplications {
		if err := validateSoftwareGroups(software.SourceIRI, software.Categories); err != nil {
			return err
		}
	}
	for _, software := range batch.SoftwareSourceCodes {
		if err := validateSoftwareGroups(software.SourceIRI, software.Categories); err != nil {
			return err
		}
	}
	return nil
}

func validateSoftwareGroups(sourceIRI string, categories []metadata.Category) error {
	if len(categories) == 0 {
		return fmt.Errorf("software %q must reference at least one category", sourceIRI)
	}
	schemes := make(map[string]struct{})
	for _, category := range categories {
		schemes[category.Scheme.SourceIRI] = struct{}{}
	}
	if len(schemes) != 1 {
		return fmt.Errorf("software %q must reference categories from exactly one scheme", sourceIRI)
	}
	return nil
}

func operationError(operation string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrDenied) || errors.Is(err, ErrConflict) || errors.Is(err, ErrInvalid) {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return fmt.Errorf("%s: %w: %w", operation, ErrInternal, err)
}
