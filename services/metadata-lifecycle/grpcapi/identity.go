package grpcapi

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/users"
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
	claims := domain.UserClaims{
		ID:         req.GetId(),
		Email:      req.GetEmail(),
		GivenName:  req.GetGivenName(),
		FamilyName: req.GetFamilyName(),
	}
	if err := claims.Normalize(); err != nil {
		return nil, err
	}

	user, err := i.users.Create(ctx, claims)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.ProvisionUserResponse{
		Id:         user.ID,
		Email:      user.Email,
		GivenName:  user.GivenName,
		FamilyName: user.FamilyName,
	}, nil
}
