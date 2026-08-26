package metadata

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDatasetValidate(t *testing.T) {
	defaultValue := int64(10)
	minimum := int64(0)
	maximum := int64(100)
	dataset := Dataset{
		ID:         uuid.New(),
		Title:      "Dataset",
		Status:     DatasetStatusDraft,
		Categories: []Category{{ID: uuid.New(), Name: "Category", Scheme: CategoryScheme{ID: uuid.New(), Name: "EPOS", Concept: TopLevelConceptData}}},
		Distributions: []Distribution{{
			ID:    uuid.New(),
			Title: "Distribution",
			WebService: &WebService{
				ID:          uuid.New(),
				Name:        "Service",
				TemplateURL: "https://example.test/{value}",
				Format:      "application/json",
				Provider:    &Organization{ID: uuid.New(), Name: "Provider"},
				Parameters: []Parameter{IntegerParameter{
					Metadata: ParameterMetadata{ID: uuid.New(), Kind: "", Variable: "value", Property: "schema:value", Label: "Value", Required: false, MultipleValues: false, ReadOnly: false},
					Minimum:  &minimum,
					Maximum:  &maximum,
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
				ID:     uuid.New(),
				Title:  "Dataset",
				Status: DatasetStatusDraft,
				Distributions: []Distribution{{
					ID:               uuid.New(),
					Title:            "Distribution",
					DownloadableFile: &DownloadableFile{URL: "https://example.test/file", Format: "text/plain"},
					WebService:       &WebService{ID: uuid.New(), Name: "Service", TemplateURL: "https://example.test", Format: "text/plain", Provider: &Organization{ID: uuid.New(), Name: "Provider"}},
				}},
			},
			want: "exactly one delivery method",
		},
		{
			name: "reversed temporal range",
			dataset: Dataset{
				ID:               uuid.New(),
				Title:            "Dataset",
				Status:           DatasetStatusDraft,
				TemporalCoverage: &Temporal{Start: &start, End: &end},
			},
			want: "start must not be after end",
		},
		{
			name: "missing category scheme",
			dataset: Dataset{
				ID:         uuid.New(),
				Title:      "Dataset",
				Status:     DatasetStatusDraft,
				Categories: []Category{{ID: uuid.New(), Name: "Category"}},
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
