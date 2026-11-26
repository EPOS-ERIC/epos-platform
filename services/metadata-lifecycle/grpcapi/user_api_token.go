package grpcapi

import (
	"context"

	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
)

func (u *UserServer) IssueUserAPIToken(ctx context.Context, _ *metadatalifecyclev1.IssueUserAPITokenRequest) (*metadatalifecyclev1.IssueUserAPITokenResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	token, err := u.users.IssueAPIToken(ctx, actor, false)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.IssueUserAPITokenResponse{Token: token}, nil
}

func (u *UserServer) RotateUserAPIToken(ctx context.Context, _ *metadatalifecyclev1.RotateUserAPITokenRequest) (*metadatalifecyclev1.RotateUserAPITokenResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	token, err := u.users.IssueAPIToken(ctx, actor, true)
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.RotateUserAPITokenResponse{Token: token}, nil
}

func (u *UserServer) RevokeUserAPIToken(ctx context.Context, request *metadatalifecyclev1.RevokeUserAPITokenRequest) (*metadatalifecyclev1.RevokeUserAPITokenResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := u.users.RevokeAPIToken(ctx, actor, request.GetUserId()); err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.RevokeUserAPITokenResponse{}, nil
}

func (u *UserServer) ValidateUserAPIToken(ctx context.Context, request *metadatalifecyclev1.ValidateUserAPITokenRequest) (*metadatalifecyclev1.ValidateUserAPITokenResponse, error) {
	userID, err := u.users.FindUserByTokenDigest(ctx, request.GetDigest())
	if err != nil {
		return nil, err
	}

	return &metadatalifecyclev1.ValidateUserAPITokenResponse{UserId: userID}, nil
}
