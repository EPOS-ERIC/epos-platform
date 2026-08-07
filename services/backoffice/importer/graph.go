package importer

import (
	"fmt"
	"io"
	"sort"

	rdf "github.com/deiu/rdf2go"
)

type graph struct {
	values map[string]map[string][]rdf.Term
}

func parseGraph(input io.Reader, baseURI string) (*graph, error) {
	parsed := rdf.NewGraph(baseURI)
	if err := parsed.Parse(input, "text/turtle"); err != nil {
		return nil, fmt.Errorf("parse turtle: %w", err)
	}

	result := &graph{values: make(map[string]map[string][]rdf.Term)}
	for triple := range parsed.IterTriples() {
		subject := triple.Subject.RawValue()
		predicate := triple.Predicate.RawValue()
		if result.values[subject] == nil {
			result.values[subject] = make(map[string][]rdf.Term)
		}
		result.values[subject][predicate] = append(result.values[subject][predicate], triple.Object)
	}
	return result, nil
}

func (g *graph) subjects(predicate, object string) []string {
	var result []string
	for subject, predicates := range g.values {
		for _, value := range predicates[predicate] {
			if value.RawValue() == object {
				result = append(result, subject)
			}
		}
	}
	sort.Strings(result)
	return result
}

func (g *graph) valuesFor(subject, predicate string) []rdf.Term {
	return g.values[subject][predicate]
}

func (g *graph) first(subject, predicate string) (rdf.Term, bool) {
	values := g.valuesFor(subject, predicate)
	if len(values) == 0 {
		return nil, false
	}
	return values[0], true
}

func (g *graph) firstString(subject, predicate string) (string, bool) {
	value, ok := g.first(subject, predicate)
	if !ok {
		return "", false
	}
	return value.RawValue(), true
}

func (g *graph) refs(subject, predicate string) []string {
	values := g.valuesFor(subject, predicate)
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.RawValue())
	}
	sort.Strings(result)
	return result
}

func (g *graph) hasType(subject, typ string) bool {
	for _, value := range g.valuesFor(subject, rdfType) {
		if value.RawValue() == typ {
			return true
		}
	}
	return false
}
