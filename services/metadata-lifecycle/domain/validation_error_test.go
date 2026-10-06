package domain

import (
	"errors"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

func TestNewDatasetInputReturnsFirstFieldFailure(t *testing.T) {
	_, err := NewDatasetInput(DatasetInput{
		Title:       " ",
		Description: "description",
	})
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
	_, err := NewDatasetDraftInput(DatasetDraftInput{
		Revision: 1,
		Metadata: DatasetInput{Description: "description"},
	})
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
	_, err := NewGroup(uuid.New(), GroupInput{Name: "name"})
	if appErr, ok := errors.AsType[*apperror.AppError](err); !ok || appErr.Code != apperror.CodeInternal {
		t.Fatalf("error = %v, want internal", err)
	}
}
