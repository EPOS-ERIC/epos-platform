package clients

import (
	"fmt"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/gateway/domain"
)

func fromProtoPreparedContactEmail(value *metadatalifecyclev1.PreparedContactEmail) (domain.ContactRequestEmail, error) {
	if value == nil {
		return domain.ContactRequestEmail{}, fmt.Errorf("prepared contact email is nil")
	}

	if value.GetTargetEmail() == "" {
		return domain.ContactRequestEmail{}, fmt.Errorf("prepared contact email target is empty")
	}

	if value.GetSubject() == "" {
		return domain.ContactRequestEmail{}, fmt.Errorf("prepared contact email subject is empty")
	}

	if value.GetBody() == "" {
		return domain.ContactRequestEmail{}, fmt.Errorf("prepared contact email body is empty")
	}

	return domain.ContactRequestEmail{
		TargetEmail: value.GetTargetEmail(),
		Subject:     value.GetSubject(),
		Body:        value.GetBody(),
	}, nil
}
