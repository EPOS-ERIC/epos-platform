package domain

import (
	"errors"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

func TestNewDatasetInputReturnsFirstFieldFailure(t *testing.T) {
	concept := ConceptInput{Name: "  test  "}
	if err := concept.Normalize(); err != nil || concept.Name != "test" {
		t.Fatalf("normalized concept = %+v, error = %v", concept, err)
	}
	concept.Name = " "
	if err := concept.Normalize(); !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatalf("blank concept error = %v, want validation_failed", err)
	}

	input := DatasetInput{
		Title:       " ",
		Description: "description",
	}
	err := input.Normalize()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %T, want *apperror.AppError", err)
	}
	if appErr.Code != apperror.CodeValidationFailed || appErr.Target != "title" {
		t.Fatalf("AppError = %+v", appErr)
	}
	if !apperror.IsCode(err, apperror.CodeValidationFailed) {
		t.Fatal("validation error has wrong application code")
	}
}

func TestNewDatasetDraftInputPrefixesMetadataTarget(t *testing.T) {
	input := DatasetDraftInput{
		Revision: 1,
		Metadata: DatasetInput{Description: "description"},
	}
	err := input.Normalize()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %T, want *apperror.AppError", err)
	}
	if appErr.Target != "metadata.title" {
		t.Fatalf("Target = %q, want metadata.title", appErr.Target)
	}
}

func TestValidateUUIDListFieldTargetsFirstRepeatedValue(t *testing.T) {
	id := uuid.New()
	err := validateUUIDListField([]uuid.UUID{id, id}, "categories", "Category")
	appErr, ok := errors.AsType[*apperror.AppError](err)
	if !ok || appErr.Code != apperror.CodeValidationFailed || appErr.Target != "categories[1]" {
		t.Fatalf("error = %+v, want validation_failed target categories[1]", err)
	}
}

func TestNewGroupKeepsPersistedValidationInternal(t *testing.T) {
	group := Group{ID: uuid.New(), Name: "name"}
	err := group.Normalize()
	if appErr, ok := errors.AsType[*apperror.AppError](err); !ok || appErr.Code != apperror.CodeInternal {
		t.Fatalf("error = %v, want internal", err)
	}
	candidate := DiscoveryCandidate{
		Type: Dataset, ResourceID: uuid.New(), VersionID: uuid.New(), GroupID: uuid.New(),
		Name: "dataset", Status: Published,
	}
	if err := candidate.Normalize(); !apperror.IsCode(err, apperror.CodeInternal) {
		t.Fatalf("missing persisted concept ID error = %v, want internal", err)
	}
	organizations := []OrganizationMetadata{{ID: uuid.New(), Name: "  Provider  "}}
	if err := validateOrganizations(organizations, "provider"); err != nil {
		t.Fatalf("normalize persisted organization: %v", err)
	}
	if organizations[0].Name != "Provider" {
		t.Fatalf("persisted organization name = %q, want Provider", organizations[0].Name)
	}
}
