package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestRenderOptionalCQLTemplate(t *testing.T) {
	name := Parameter{ID: uuid.New(), ParameterInput: ParameterInput{Name: "name"}}
	enabled := Parameter{ID: uuid.New(), ParameterInput: ParameterInput{Name: "enabled"}}
	limit := Parameter{ID: uuid.New(), ParameterInput: ParameterInput{Name: "limit"}}
	parameters := []Parameter{name, enabled, limit}
	const source = `WHERE 1 = 1{{with index . "name"}} AND name = {{cql .}}{{end}}{{with index . "enabled"}} AND enabled = {{cql .}}{{end}}{{with index . "limit"}} AND limit = {{cql .}}{{end}}`

	tests := []struct {
		name    string
		values  accessValues
		want    string
		omitted bool
	}{
		{
			name: "omits when no child is supplied",
			values: accessValues{
				resolved: map[uuid.UUID]AccessArray{},
				supplied: map[uuid.UUID]bool{},
			},
			omitted: true,
		},
		{
			name: "renders one supplied child",
			values: accessValues{
				resolved: map[uuid.UUID]AccessArray{limit.ID: {AccessInteger(10)}},
				supplied: map[uuid.UUID]bool{limit.ID: true},
			},
			want: "WHERE 1 = 1 AND limit = 10",
		},
		{
			name: "composes supplied children and escapes strings",
			values: accessValues{
				resolved: map[uuid.UUID]AccessArray{
					name.ID:    {AccessString("O'Brien")},
					enabled.ID: {AccessBoolean(true)},
					limit.ID:   {AccessInteger(10)},
				},
				supplied: map[uuid.UUID]bool{name.ID: true, enabled.ID: true, limit.ID: true},
			},
			want: "WHERE 1 = 1 AND name = 'O''Brien' AND enabled = true AND limit = 10",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered, err := test.values.renderTemplate(source, parameters, false)
			if err != nil {
				t.Fatalf("renderTemplate() error = %v", err)
			}
			if test.omitted {
				if rendered != nil {
					t.Fatalf("renderTemplate() = %q, want omitted", *rendered)
				}
				return
			}
			if rendered == nil || *rendered != test.want {
				t.Fatalf("renderTemplate() = %v, want %q", rendered, test.want)
			}
		})
	}
}

func TestParseTemplateDefinitionRecognizesCQL(t *testing.T) {
	_, _, err := parseTemplateDefinition(`{{with index . "value"}}{{cql .}}{{end}}`, []ParameterInput{{Name: "value", Label: "Value", Schema: StringSchema{}}})
	if err != nil {
		t.Fatalf("parseTemplateDefinition() error = %v", err)
	}
}
