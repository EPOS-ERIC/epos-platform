package grpcapi

import (
	"context"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/contactrequests"
)

type ContactRequestServer struct {
	metadatalifecyclev1.UnimplementedContactRequestServiceServer
	contactRequests *contactrequests.Service
}

func NewContactRequest(contactRequests *contactrequests.Service) *ContactRequestServer {
	return &ContactRequestServer{contactRequests: contactRequests}
}

func (s *ContactRequestServer) PrepareEmails(ctx context.Context, request *metadatalifecyclev1.PrepareEmailsRequest) (*metadatalifecyclev1.PrepareEmailsResponse, error) {
	if request == nil {
		return nil, apperror.New(apperror.CodeValidationFailed, "contact request is required")
	}

	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	target, err := toDomainContactRequestTarget(request)
	if err != nil {
		return nil, err
	}

	emails, err := s.contactRequests.Prepare(ctx, actor, target, request.GetMessage())
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.PrepareEmailsResponse{Emails: toProtoPreparedContactEmails(emails)}, nil
}
