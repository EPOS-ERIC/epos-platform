package domain

import (
	"fmt"
	"math"
	"regexp"

	"github.com/google/uuid"
)

func (d Dataset) Validate() error {
	if d.ID == uuid.Nil {
		return fmt.Errorf("dataset ID is required")
	}
	if d.Title == "" {
		return fmt.Errorf("dataset title is required")
	}
	if !d.Status.Valid() {
		return fmt.Errorf("dataset status %q is invalid", d.Status)
	}
	if d.OriginalDatasetID == uuid.Nil {
		return fmt.Errorf("original dataset ID is required")
	}
	if err := validateSpatial(d.SpatialCoverage); err != nil {
		return fmt.Errorf("dataset spatial coverage: %w", err)
	}
	if err := validateTemporal(d.TemporalCoverage); err != nil {
		return fmt.Errorf("dataset temporal coverage: %w", err)
	}
	if d.Editor != nil {
		if err := validateUser(*d.Editor); err != nil {
			return fmt.Errorf("dataset editor: %w", err)
		}
	}
	if d.Group != nil {
		if err := validateGroup(*d.Group); err != nil {
			return fmt.Errorf("dataset group: %w", err)
		}
	}

	identifiers := make(map[string]struct{}, len(d.PersistentIdentifiers))
	for _, identifier := range d.PersistentIdentifiers {
		if err := validateIdentifier(identifier); err != nil {
			return fmt.Errorf("dataset identifier: %w", err)
		}
		if err := addUnique(identifiers, identifier.ID.String(), "dataset identifier"); err != nil {
			return err
		}
	}
	contacts := make(map[string]struct{}, len(d.ContactPoints))
	for _, contact := range d.ContactPoints {
		if err := validateContactPoint(contact); err != nil {
			return fmt.Errorf("dataset contact point: %w", err)
		}
		if err := addUnique(contacts, contact.ID.String(), "dataset contact point"); err != nil {
			return err
		}
	}
	providers := make(map[string]struct{}, len(d.Providers))
	for _, provider := range d.Providers {
		if err := validateOrganization(provider); err != nil {
			return fmt.Errorf("dataset provider: %w", err)
		}
		if err := addUnique(providers, provider.ID.String(), "dataset provider"); err != nil {
			return err
		}
	}
	categories := make(map[string]struct{}, len(d.Categories))
	for _, category := range d.Categories {
		if err := validateCategory(category); err != nil {
			return fmt.Errorf("dataset category: %w", err)
		}
		if err := addUnique(categories, category.ID.String(), "dataset category"); err != nil {
			return err
		}
	}
	distributions := make(map[string]struct{}, len(d.Distributions))
	for _, distribution := range d.Distributions {
		if err := validateDistribution(distribution); err != nil {
			return fmt.Errorf("dataset distribution: %w", err)
		}
		if err := addUnique(distributions, distribution.ID.String(), "dataset distribution"); err != nil {
			return err
		}
	}

	return nil
}

func validateCategory(c Category) error {
	if c.ID == uuid.Nil || c.Name == "" {
		return fmt.Errorf("ID and name are required")
	}
	if err := validateCategoryScheme(c.Scheme); err != nil {
		return fmt.Errorf("scheme: %w", err)
	}
	return nil
}

func validateCategoryScheme(s CategoryScheme) error {
	if s.ID == uuid.Nil || s.Name == "" || !s.Concept.Valid() {
		return fmt.Errorf("ID, name, and valid concept are required")
	}
	return nil
}

func validateDistribution(d Distribution) error {
	if d.ID == uuid.Nil || d.Title == "" {
		return fmt.Errorf("ID and title are required")
	}
	if (d.WebService == nil) == (d.DownloadableFile == nil) {
		return fmt.Errorf("exactly one delivery method is required")
	}
	if d.DownloadableFile != nil && (d.DownloadableFile.URL == "" || d.DownloadableFile.Format == "") {
		return fmt.Errorf("downloadable file URL and format are required")
	}
	if d.WebService != nil {
		if err := validateWebService(*d.WebService); err != nil {
			return fmt.Errorf("web service: %w", err)
		}
	}
	return nil
}

func validateWebService(w WebService) error {
	if w.ID == uuid.Nil || w.Name == "" || w.TemplateURL == "" || w.Format == "" {
		return fmt.Errorf("ID, name, template URL, and format are required")
	}
	if err := validateSpatial(w.SpatialCoverage); err != nil {
		return fmt.Errorf("spatial coverage: %w", err)
	}
	if err := validateTemporal(w.TemporalCoverage); err != nil {
		return fmt.Errorf("temporal coverage: %w", err)
	}
	if w.Documentation != nil && w.Documentation.URI == "" {
		return fmt.Errorf("documentation URI is required when documentation is present")
	}
	providers := make(map[string]struct{}, len(w.Providers))
	for _, provider := range w.Providers {
		if err := validateOrganization(provider); err != nil {
			return fmt.Errorf("provider: %w", err)
		}
		if err := addUnique(providers, provider.ID.String(), "provider"); err != nil {
			return err
		}
	}
	contacts := make(map[string]struct{}, len(w.ContactPoints))
	for _, contact := range w.ContactPoints {
		if err := validateContactPoint(contact); err != nil {
			return fmt.Errorf("contact point: %w", err)
		}
		if err := addUnique(contacts, contact.ID.String(), "contact point"); err != nil {
			return err
		}
	}
	variables := make(map[string]struct{}, len(w.Parameters))
	for _, parameter := range w.Parameters {
		if err := validateParameter(parameter); err != nil {
			return fmt.Errorf("parameter %q: %w", parameter.GetMetadata().SourceIRI, err)
		}
		if err := addUnique(variables, parameter.GetMetadata().Variable, "parameter variable"); err != nil {
			return err
		}
	}
	return nil
}

func validateUser(u User) error {
	if u.ID == uuid.Nil || u.AuthIdentifier == "" || !u.Status.Valid() {
		return fmt.Errorf("ID, auth identifier, and valid status are required")
	}
	return nil
}

func validateGroup(g Group) error {
	if g.ID == uuid.Nil || g.Slug == "" || g.Name == "" {
		return fmt.Errorf("ID, slug, and name are required")
	}
	return nil
}

func validateOrganization(o Organization) error {
	if o.ID == uuid.Nil || o.Name == "" {
		return fmt.Errorf("ID and name are required")
	}
	identifiers := make(map[string]struct{}, len(o.Identifiers))
	for _, identifier := range o.Identifiers {
		if err := validateIdentifier(identifier); err != nil {
			return fmt.Errorf("identifier: %w", err)
		}
		if err := addUnique(identifiers, identifier.ID.String(), "organization identifier"); err != nil {
			return err
		}
	}
	return nil
}

func validateIdentifier(i Identifier) error {
	if i.ID == uuid.Nil || i.Type == "" || i.Value == "" {
		return fmt.Errorf("ID, type, and value are required")
	}
	return nil
}

func validateContactPoint(c ContactPoint) error {
	if c.ID == uuid.Nil || c.Email == "" || c.Role == "" {
		return fmt.Errorf("ID, email, and role are required")
	}
	return nil
}

func validateSpatial(s *Spatial) error {
	if s == nil {
		return nil
	}
	if s.Point != nil {
		if err := validateCoordinate(*s.Point); err != nil {
			return fmt.Errorf("point: %w", err)
		}
	}
	if s.Polygon != nil {
		if len(s.Polygon) < 4 {
			return fmt.Errorf("polygon must contain at least four coordinates")
		}
		for _, coordinate := range s.Polygon {
			if err := validateCoordinate(coordinate); err != nil {
				return fmt.Errorf("polygon: %w", err)
			}
		}
		first, last := s.Polygon[0], s.Polygon[len(s.Polygon)-1]
		if first != last {
			return fmt.Errorf("polygon must be closed")
		}
	}
	return nil
}

func validateCoordinate(c Coordinate) error {
	if math.IsNaN(c.Latitude) || math.IsNaN(c.Longitude) || c.Latitude < -90 || c.Latitude > 90 || c.Longitude < -180 || c.Longitude > 180 {
		return fmt.Errorf("coordinate is outside WGS84 bounds")
	}
	return nil
}

func validateTemporal(t *Temporal) error {
	if t != nil && t.Start != nil && t.End != nil && t.Start.After(*t.End) {
		return fmt.Errorf("start must not be after end")
	}
	return nil
}

func validateParameter(p Parameter) error {
	if p == nil {
		return fmt.Errorf("is required")
	}
	metadata := p.GetMetadata()
	if metadata.ID == uuid.Nil || metadata.Variable == "" || metadata.Property == "" || metadata.Label == "" {
		return fmt.Errorf("ID, variable, property, and label are required")
	}
	switch value := p.(type) {
	case StringParameter:
		if err := validatePattern(value.Pattern); err != nil {
			return err
		}
		if value.Default != nil && len(value.AllowedValues) > 0 && !contains(value.AllowedValues, *value.Default) {
			return fmt.Errorf("default is not an allowed value")
		}
	case BooleanParameter:
	case IntegerParameter:
		if err := validateIntegerParameter(value); err != nil {
			return err
		}
	case FloatParameter:
		if err := validateFloatParameter(value); err != nil {
			return err
		}
	case DateTimeParameter:
		if err := validatePattern(value.Pattern); err != nil {
			return err
		}
		if value.Minimum != nil && value.Maximum != nil && value.Minimum.After(*value.Maximum) {
			return fmt.Errorf("minimum must not be after maximum")
		}
		if value.Default != nil && ((value.Minimum != nil && value.Default.Before(*value.Minimum)) || (value.Maximum != nil && value.Default.After(*value.Maximum))) {
			return fmt.Errorf("default is outside allowed range")
		}
	default:
		return fmt.Errorf("unsupported type %T", p)
	}
	return nil
}

func validateIntegerParameter(p IntegerParameter) error {
	if p.Minimum != nil && p.Maximum != nil && *p.Minimum > *p.Maximum {
		return fmt.Errorf("minimum must not exceed maximum")
	}
	if p.Default != nil && ((p.Minimum != nil && *p.Default < *p.Minimum) || (p.Maximum != nil && *p.Default > *p.Maximum)) {
		return fmt.Errorf("default is outside allowed range")
	}
	if p.Default != nil && len(p.AllowedValues) > 0 && !contains(p.AllowedValues, *p.Default) {
		return fmt.Errorf("default is not an allowed value")
	}
	return nil
}

func validateFloatParameter(p FloatParameter) error {
	if (p.Minimum != nil && !isFinite(*p.Minimum)) || (p.Maximum != nil && !isFinite(*p.Maximum)) || (p.Default != nil && !isFinite(*p.Default)) {
		return fmt.Errorf("values must be finite")
	}
	if p.Minimum != nil && p.Maximum != nil && *p.Minimum > *p.Maximum {
		return fmt.Errorf("minimum must not exceed maximum")
	}
	if p.Default != nil && ((p.Minimum != nil && *p.Default < *p.Minimum) || (p.Maximum != nil && *p.Default > *p.Maximum)) {
		return fmt.Errorf("default is outside allowed range")
	}
	if p.Default != nil && len(p.AllowedValues) > 0 && !contains(p.AllowedValues, *p.Default) {
		return fmt.Errorf("default is not an allowed value")
	}
	return nil
}

func validatePattern(pattern string) error {
	if pattern == "" {
		return nil
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}
	return nil
}

func addUnique(seen map[string]struct{}, value, name string) error {
	if _, exists := seen[value]; exists {
		return fmt.Errorf("duplicate %s %q", name, value)
	}
	seen[value] = struct{}{}
	return nil
}

func contains[T comparable](values []T, value T) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
