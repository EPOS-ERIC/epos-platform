package grpcapi

import (
	"context"
	"errors"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type IdentityServer struct {
	metadatalifecyclev1.UnimplementedIdentityServiceServer
	users *users.Service
}

func NewIdentity(users *users.Service) *IdentityServer {
	return &IdentityServer{
		users: users,
	}
}

func (i *IdentityServer) ProvisionUser(ctx context.Context, req *metadatalifecyclev1.ProvisionUserRequest) (*metadatalifecyclev1.ProvisionUserResponse, error) {
	claims, err := domain.NewUserClaims(domain.UserClaims{
		ID:         req.GetId(),
		Email:      req.GetEmail(),
		GivenName:  req.GetGivenName(),
		FamilyName: req.GetFamilyName(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user claims")
	}

	user, err := i.users.Create(ctx, claims)
	if err != nil {
		return nil, userStatusError(err)
	}

	return &metadatalifecyclev1.ProvisionUserResponse{
		Id:         user.ID,
		Email:      user.Email,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
	}, nil
}

func userStatusError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidUser):
		return status.Error(codes.InvalidArgument, "invalid user claims")
	case errors.Is(err, domain.ErrUserConflict):
		return status.Error(codes.AlreadyExists, "user already exists")
	default:
		return status.Error(codes.Internal, "provision user")
	}
}
