package metadataimport

import (
	"errors"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/backoffice/identity"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/metadata"
	"github.com/EPOS-ERIC/epos-platform/services/backoffice/turtle"
)

func TestApplyStatusDefaultsToDraft(t *testing.T) {
	batch := turtle.Batch{Datasets: []metadata.Dataset{{}}}

	if err := applyStatus(&batch, identity.User{}, ""); err != nil {
		t.Fatalf("applyStatus() error = %v", err)
	}
	if got := batch.Datasets[0].Status; got != metadata.DatasetStatusDraft {
		t.Fatalf("dataset status = %q, want %q", got, metadata.DatasetStatusDraft)
	}
}

func TestApplyStatusAllowsNonDraftForSuperAdmin(t *testing.T) {
	batch := turtle.Batch{Datasets: []metadata.Dataset{{}}}
	editor := identity.User{IsSuperAdmin: true}

	if err := applyStatus(&batch, editor, metadata.DatasetStatusPublished); err != nil {
		t.Fatalf("applyStatus() error = %v", err)
	}
	if got := batch.Datasets[0].Status; got != metadata.DatasetStatusPublished {
		t.Fatalf("dataset status = %q, want %q", got, metadata.DatasetStatusPublished)
	}
}

func TestApplyStatusRejectsNonDraftForRegularUser(t *testing.T) {
	batch := turtle.Batch{Datasets: []metadata.Dataset{{}}}

	err := applyStatus(&batch, identity.User{}, metadata.DatasetStatusPublished)
	if !errors.Is(err, ErrMetadataImportStatus) {
		t.Fatalf("applyStatus() error = %v, want %v", err, ErrMetadataImportStatus)
	}
}
