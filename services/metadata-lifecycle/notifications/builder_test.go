package notifications

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
)

type conversionStore struct {
	Store
	t *testing.T
}

type conversionAuthorizer struct{ Authorizer }

func (s conversionStore) GetNotificationUserContext(_ context.Context, id string) (domain.UserSummary, error) {
	return domain.UserSummary{ID: id, GivenName: "Preview", FamilyName: "User", Email: "preview@example.org"}, nil
}

func (s conversionStore) GetNotificationGroupContext(_ context.Context, _ uuid.UUID) (domain.NotificationGroupContext, error) {
	return domain.NotificationGroupContext{Name: "Preview group", Concept: "DATA"}, nil
}

func (s conversionStore) GlobalRecipients(_ context.Context, topic domain.NotificationTopicKey) ([]domain.NotificationRecipient, error) {
	if topic != domain.NotificationTopicGlobalConversionRequested {
		s.t.Fatalf("GlobalRecipients() topic = %q", topic)
	}
	return []domain.NotificationRecipient{{ID: "recipient-1", Email: "recipient@example.org", Status: domain.UserStatusActive}}, nil
}

func TestBuildConversionRequested(t *testing.T) {
	event := domain.ConversionRequested{
		ActorID: "requester-1",
		Resource: domain.NotificationResource{
			Type:       domain.Dataset,
			Name:       "Example dataset",
			ResourceID: uuid.MustParse("11111111-1111-4111-8111-111111111111"),
			VersionID:  uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		},
		DistributionID:    uuid.MustParse("33333333-3333-4333-8333-333333333333"),
		DistributionTitle: "Example distribution",
		WebServiceType:    domain.WebServiceTypeHTTP,
		Input: domain.ConversionRequestInput{
			InputFormat:    "application/json",
			Visualizations: []domain.WebServiceVisibleOn{domain.WebServiceVisibleOnGraph, domain.WebServiceVisibleOnMap},
			Message:        "Please convert this dataset",
		},
		DefaultURL: "https://example.org/data",
	}

	s := New(slog.New(slog.NewTextHandler(io.Discard, nil)), conversionStore{t: t}, conversionAuthorizer{}, "https://example.org/backoffice/")
	got, err := s.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Build() returned %d notifications, want 1", len(got))
	}
	if got[0].TargetUserID != "recipient-1" || got[0].TargetEmail != "recipient@example.org" || got[0].EventKind != domain.NotificationEventConversionRequested || got[0].Channel != domain.NotificationChannelEmail || got[0].Subject != "Conversion requested: Example dataset" {
		t.Errorf("unexpected notification: %+v", got[0])
	}

	for _, want := range []string{
		"Conversion requested: Example dataset",
		"Preview User (preview@example.org, user ID: requester-1)",
		"Example distribution",
		"Distribution ID: 33333333-3333-4333-8333-333333333333",
		"Dataset version ID:",
		"Requested visualizations:",
		"Input format to use:",
		"Service default URL:",
		"application/json",
		"GRAPH, MAP",
		"https://example.org/data",
		`href="https://example.org/backoffice/data/11111111-1111-4111-8111-111111111111/22222222-2222-4222-8222-222222222222"`,
		`href="https://example.org/backoffice/data/11111111-1111-4111-8111-111111111111/22222222-2222-4222-8222-222222222222#distributions"`,
		"Please convert this dataset",
	} {
		if !strings.Contains(got[0].Body, want) {
			t.Errorf("notification body missing %q", want)
		}
	}

	groupID := uuid.New()
	body, err := s.render(context.Background(), domain.MembershipRequested{SubjectID: "requester-1", GroupID: groupID})
	if err != nil {
		t.Fatalf("render membership request: %v", err)
	}
	for _, want := range []string{"Preview User (preview@example.org, user ID: requester-1)", "Preview group (concept: DATA, group ID: " + groupID.String() + ")"} {
		if !strings.Contains(body, want) {
			t.Errorf("membership body missing %q", want)
		}
	}
}
