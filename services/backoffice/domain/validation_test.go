package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDatasetValidate(t *testing.T) {
	defaultValue := int64(10)
	dataset := Dataset{
		ID:                uuid.New(),
		Title:             "Dataset",
		Status:            DatasetStatusDraft,
		OriginalDatasetID: uuid.New(),
		Editor: userPtr(User{
			ID:             uuid.New(),
			AuthIdentifier: "editor",
			Status:         UserStatusActive,
		}),
		Group: groupPtr(Group{
			ID:   uuid.New(),
			Slug: "group",
			Name: "Group",
		}),
		Categories: []Category{{ID: uuid.New(), Name: "Category", Scheme: CategoryScheme{ID: uuid.New(), Name: "EPOS", Concept: TopLevelConceptData}}},
		Distributions: []Distribution{{
			ID:    uuid.New(),
			Title: "Distribution",
			WebService: &WebService{
				ID:          uuid.New(),
				Name:        "Service",
				TemplateURL: "https://example.test/{value}",
				Format:      "application/json",
				Parameters: []Parameter{IntegerParameter{
					Metadata: ParameterMetadata{ID: uuid.New(), Variable: "value", Property: "schema:value", Label: "Value"},
					Minimum:  int64Ptr(0),
					Maximum:  int64Ptr(100),
					Default:  &defaultValue,
				}},
			},
		}},
	}

	if err := dataset.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestDatasetValidateRejectsInvalidNestedValues(t *testing.T) {
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(-time.Hour)

	tests := []struct {
		name    string
		dataset Dataset
		want    string
	}{
		{
			name: "both delivery methods",
			dataset: Dataset{
				ID:                uuid.New(),
				Title:             "Dataset",
				Status:            DatasetStatusDraft,
				OriginalDatasetID: uuid.New(),
				Distributions: []Distribution{{
					ID:               uuid.New(),
					Title:            "Distribution",
					DownloadableFile: &DownloadableFile{URL: "https://example.test/file", Format: "text/plain"},
					WebService:       &WebService{ID: uuid.New(), Name: "Service", TemplateURL: "https://example.test", Format: "text/plain"},
				}},
			},
			want: "exactly one delivery method",
		},
		{
			name: "reversed temporal range",
			dataset: Dataset{
				ID:                uuid.New(),
				Title:             "Dataset",
				Status:            DatasetStatusDraft,
				OriginalDatasetID: uuid.New(),
				TemporalCoverage:  &Temporal{Start: &start, End: &end},
			},
			want: "start must not be after end",
		},
		{
			name: "missing category scheme",
			dataset: Dataset{
				ID:                uuid.New(),
				Title:             "Dataset",
				Status:            DatasetStatusDraft,
				OriginalDatasetID: uuid.New(),
				Categories:        []Category{{ID: uuid.New(), Name: "Category"}},
			},
			want: "scheme",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.dataset.Validate()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestParameterRecordRoundTrip(t *testing.T) {
	parameter := StringParameter{
		Metadata:      ParameterMetadata{ID: uuid.New(), Variable: "format", Property: "schema:format", Label: "Format"},
		Default:       stringPtr("json"),
		AllowedValues: []string{"json", "csv"},
	}

	record, err := EncodeParameter(parameter)
	if err != nil {
		t.Fatalf("EncodeParameter() error = %v", err)
	}
	decoded, err := DecodeParameter(record)
	if err != nil {
		t.Fatalf("DecodeParameter() error = %v", err)
	}
	stringParameter, ok := decoded.(StringParameter)
	if !ok {
		t.Fatalf("DecodeParameter() type = %T, want StringParameter", decoded)
	}
	if stringParameter.Kind() != ParameterKindString || *stringParameter.Default != "json" {
		t.Fatalf("DecodeParameter() = %#v", stringParameter)
	}
}

func userPtr(user User) *User { return &user }

func groupPtr(group Group) *Group { return &group }

func int64Ptr(value int64) *int64 { return &value }

func stringPtr(value string) *string { return &value }
