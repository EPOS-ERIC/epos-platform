package grpcapi

import (
	"context"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	metadatalifecyclev1 "github.com/EPOS-ERIC/epos-platform/packages/protobuf/gen/go/metadata_lifecycle/v1"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/organizations"
	"github.com/google/uuid"
)

type OrganizationServer struct {
	metadatalifecyclev1.UnimplementedOrganizationServiceServer
	organizations *organizations.Service
}

func NewOrganization(organizations *organizations.Service) *OrganizationServer {
	return &OrganizationServer{organizations: organizations}
}

func (o *OrganizationServer) ListOrganizations(ctx context.Context, _ *metadatalifecyclev1.ListOrganizationsRequest) (*metadatalifecyclev1.ListOrganizationsResponse, error) {
	values, err := o.organizations.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*metadatalifecyclev1.Organization, len(values))
	for i, value := range values {
		result[i] = toProtoOrganization(value)
	}
	return &metadatalifecyclev1.ListOrganizationsResponse{Organizations: result}, nil
}

func (o *OrganizationServer) GetOrganization(ctx context.Context, request *metadatalifecyclev1.GetOrganizationRequest) (*metadatalifecyclev1.GetOrganizationResponse, error) {
	id, err := parseOrganizationID(request.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	value, err := o.organizations.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.GetOrganizationResponse{Organization: toProtoOrganization(value)}, nil
}

func (o *OrganizationServer) CreateOrganization(ctx context.Context, request *metadatalifecyclev1.CreateOrganizationRequest) (*metadatalifecyclev1.CreateOrganizationResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	input, err := toOrganizationInput(request.Name, request.Street, request.Locality, request.PostalCode, request.Country, request.LogoUrl, request.PageUrl, request.Email, request.LeiCode, request.Identifiers, request.Parents, request.ContactPointIds)
	if err != nil {
		return nil, err
	}
	value, err := o.organizations.Create(ctx, actor, input)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.CreateOrganizationResponse{Organization: toProtoOrganization(value)}, nil
}

func (o *OrganizationServer) ReplaceOrganization(ctx context.Context, request *metadatalifecyclev1.ReplaceOrganizationRequest) (*metadatalifecyclev1.ReplaceOrganizationResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseOrganizationID(request.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	input, err := toOrganizationInput(request.Name, request.Street, request.Locality, request.PostalCode, request.Country, request.LogoUrl, request.PageUrl, request.Email, request.LeiCode, request.Identifiers, request.Parents, request.ContactPointIds)
	if err != nil {
		return nil, err
	}
	value, err := o.organizations.Replace(ctx, actor, id, input)
	if err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.ReplaceOrganizationResponse{Organization: toProtoOrganization(value)}, nil
}

func (o *OrganizationServer) DeleteOrganization(ctx context.Context, request *metadatalifecyclev1.DeleteOrganizationRequest) (*metadatalifecyclev1.DeleteOrganizationResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseOrganizationID(request.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	if err := o.organizations.Delete(ctx, actor, id); err != nil {
		return nil, err
	}
	return &metadatalifecyclev1.DeleteOrganizationResponse{}, nil
}

func parseOrganizationID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.New(apperror.CodeValidationFailed, "invalid organization id")
	}
	return id, nil
}

func toOrganizationInput(name string, street, locality, postalCode, country, logo, page, email, lei *string, identifiers []*metadatalifecyclev1.IdentifierInput, parents, contactPoints []string) (domain.OrganizationInput, error) {
	input := domain.OrganizationInput{
		Name:        name,
		Street:      street,
		Locality:    locality,
		PostalCode:  postalCode,
		Country:     country,
		LogoURL:     logo,
		PageURL:     page,
		Email:       email,
		LEICode:     lei,
		Identifiers: make([]domain.OrganizationIdentifierInput, len(identifiers)),
	}
	for i, identifier := range identifiers {
		if identifier == nil {
			return domain.OrganizationInput{}, apperror.Field(fmt.Sprintf("identifiers[%d]", i), "Identifier is required.", fmt.Errorf("missing identifier at index %d", i))
		}
		input.Identifiers[i] = domain.OrganizationIdentifierInput{Type: identifier.Type, Value: identifier.Value}
	}
	var err error
	input.Parents, err = parseUUIDs(parents, "parents")
	if err != nil {
		return domain.OrganizationInput{}, err
	}
	input.ContactPoints, err = parseUUIDs(contactPoints, "contact_point_ids")
	if err != nil {
		return domain.OrganizationInput{}, err
	}
	result, err := domain.NewOrganizationInput(input)
	if err != nil {
		return domain.OrganizationInput{}, fmt.Errorf("validate organization input: %w", err)
	}
	return result, nil
}

func parseUUIDs(values []string, target string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, len(values))
	for i, value := range values {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			if err == nil {
				err = fmt.Errorf("%s must not be the zero UUID", target)
			}
			return nil, apperror.Field(fmt.Sprintf("%s[%d]", target, i), "Must be a valid UUID.", fmt.Errorf("parse %s: %w", target, err))
		}
		result[i] = id
	}
	return result, nil
}

func toProtoOrganization(value domain.Organization) *metadatalifecyclev1.Organization {
	result := &metadatalifecyclev1.Organization{Id: value.ID.String(), Name: value.Name, Street: value.Street, Locality: value.Locality, PostalCode: value.PostalCode, Country: value.Country, LogoUrl: value.LogoURL, PageUrl: value.PageURL, Email: value.Email, LeiCode: value.LEICode}
	result.Parents = uuidStrings(value.Parents)
	result.ContactPointIds = uuidStrings(value.ContactPoints)
	result.Identifiers = make([]*metadatalifecyclev1.Identifier, len(value.Identifiers))
	for i, identifier := range value.Identifiers {
		result.Identifiers[i] = &metadatalifecyclev1.Identifier{Id: identifier.ID.String(), Type: identifier.Type, Value: identifier.Value}
	}
	return result
}

func uuidStrings(values []uuid.UUID) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.String()
	}
	return result
}
