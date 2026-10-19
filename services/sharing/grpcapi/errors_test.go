package grpcapi

import (
	"testing"

	"buf.build/go/protovalidate"
	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	configurationv1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/configuration/v1"
)

func TestValidationErrorKeepsFieldTarget(t *testing.T) {
	validator, err := protovalidate.New()
	if err != nil {
		t.Fatal(err)
	}
	err = validator.Validate(&configurationv1.GetConfigurationRequest{Id: "not-a-uuid"})
	if err == nil {
		t.Fatal("expected protobuf validation failure")
	}
	appErr := ValidationAppError(err)
	if appErr.Code != apperror.CodeValidationFailed || appErr.Target != "id" {
		t.Fatalf("validation error = %+v, want validation_failed at id", appErr)
	}
}
