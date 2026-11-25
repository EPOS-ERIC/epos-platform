package domain

import (
	"fmt"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
)

var (
	wfsGeneratedParameters = reservedParameterNames(
		"SERVICE", "REQUEST", "VERSION", "TYPENAME", "TYPENAMES", "OUTPUTFORMAT", "MAXFEATURES", "COUNT", "BBOX", "SRSNAME",
	)
	wfsReservedParameters = reservedParameterNames(
		"SERVICE", "REQUEST", "VERSION", "ACCEPTVERSIONS", "ACCEPTFORMATS", "SECTIONS", "UPDATESEQUENCE",
		"TYPENAME", "TYPENAMES", "OUTPUTFORMAT", "MAXFEATURES", "COUNT", "BBOX", "SRSNAME",
		"PROPERTYNAME", "FEATUREVERSION", "FEATUREID", "RESOURCEID", "FILTER", "FILTER_LANGUAGE", "SORTBY", "RESULTTYPE", "STARTINDEX",
		"NAMESPACE", "NAMESPACES", "ALIASES", "RESOLVE", "RESOLVEDEPTH", "RESOLVETIMEOUT", "STOREDQUERY_ID",
	)
)

func validateWFSChoices(values []string, selected, field, defaultField string) error {
	if len(values) == 0 {
		return apperror.Field(field, "Configure at least one choice.", nil)
	}
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		target := fmt.Sprintf("%s[%d]", field, i)
		if value == "" || strings.TrimSpace(value) != value {
			return apperror.Field(target, "Choices must not be blank or contain surrounding whitespace.", nil)
		}
		if _, exists := seen[value]; exists {
			return apperror.Field(target, "Choices must be unique.", nil)
		}
		seen[value] = struct{}{}
	}
	if _, exists := seen[selected]; !exists {
		return apperror.Field(defaultField, "Default must match a configured choice.", nil)
	}
	return nil
}

// ############### INPUT ###############

type WFSWebServiceInput struct {
	WebServiceCommonInput
	GetFeatureTemplateURL string
	Version               string
	FeatureTypes          []string
	DefaultFeatureType    string
	OutputFormats         []string
	DefaultOutputFormat   string
	ResultLimit           *int64
	ResultLimitEditable   bool
	Parameters            []WebServiceParameterInput
}

func (*WFSWebServiceInput) ServiceType() WebServiceType {
	return WebServiceTypeWFS
}

func (value *WFSWebServiceInput) Normalize() error {
	if value == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	if err := value.WebServiceCommonInput.Normalize(); err != nil {
		return err
	}
	value.GetFeatureTemplateURL = strings.TrimSpace(value.GetFeatureTemplateURL)
	if value.Version != "1.1.0" && value.Version != "2.0.0" {
		return apperror.Field("version", "Select WFS version 1.1.0 or 2.0.0.", nil)
	}
	if err := validateWFSChoices(value.FeatureTypes, value.DefaultFeatureType, "feature_types", "default_feature_type"); err != nil {
		return err
	}
	if err := validateWFSChoices(value.OutputFormats, value.DefaultOutputFormat, "output_formats", "default_output_format"); err != nil {
		return err
	}
	if value.ResultLimit != nil && *value.ResultLimit <= 0 {
		return apperror.Field("result_limit", "Result limit must be positive.", nil)
	}
	parameters, err := parseWebServiceParameterInputs(value.Parameters, "parameters")
	if err != nil {
		return err
	}
	if err := validateWFSParameters(value.GetFeatureTemplateURL, parameters); err != nil {
		return err
	}
	value.Parameters = parameters
	return nil
}

func validateWFSParameters(templateURL string, parameters []WebServiceParameterInput) error {
	if err := validateWebServiceParameterRoles(WebServiceTypeWFS, parameters, "parameters", nil); err != nil {
		return err
	}
	for i, parameter := range parameters {
		target := fmt.Sprintf("parameters[%d]", i)
		if parameter.Location != ParameterLocationPath && parameter.Location != ParameterLocationQuery {
			return apperror.Field(target+".location", "WFS parameters must use PATH or QUERY location.", nil)
		}
		if _, reserved := wfsReservedParameters[strings.ToUpper(parameter.Name)]; reserved {
			return apperror.Field(target+".name", "Standard WFS parameters cannot be redefined.", nil)
		}
	}
	endpoint, err := url.Parse(templateURL)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Fragment != "" {
		return apperror.Field("get_feature_template_url", "URL must be an absolute HTTP(S) URL without a fragment.", err)
	}
	if pathPlaceholderPattern.MatchString(endpoint.Host) || pathPlaceholderPattern.MatchString(endpoint.RawQuery) {
		return apperror.Field("get_feature_template_url", "Placeholders are only supported in the URL path.", nil)
	}
	if err := validatePathParameters(templateURL, pathParameterNames(parameters)); err != nil {
		return apperror.Field("get_feature_template_url", "Template URL placeholders must match declared path parameters.", err)
	}
	query, err := url.ParseQuery(endpoint.RawQuery)
	if err != nil {
		return apperror.Field("get_feature_template_url", "URL query must be valid.", err)
	}
	for name := range query {
		if _, reserved := wfsGeneratedParameters[strings.ToUpper(name)]; reserved {
			return apperror.Field("get_feature_template_url", "Platform-generated WFS parameters must not appear in the URL query.", nil)
		}
	}
	return nil
}

// ############### END INPUT ###############

type WFSWebService struct {
	WebServiceCommon
	GetFeatureTemplateURL string
	Version               string
	FeatureTypes          []string
	DefaultFeatureType    string
	OutputFormats         []string
	DefaultOutputFormat   string
	ResultLimit           *int64
	ResultLimitEditable   bool
	Parameters            []WebServiceParameter
}

func (w *WFSWebService) ServiceType() WebServiceType {
	return WebServiceTypeWFS
}

func (w *WFSWebService) Normalize() (err error) {
	if w == nil {
		return apperror.Default(apperror.CodeInternal)
	}
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	if err := w.WebServiceCommon.Normalize(); err != nil {
		return err
	}
	w.GetFeatureTemplateURL = strings.TrimSpace(w.GetFeatureTemplateURL)
	if w.Version != "1.1.0" && w.Version != "2.0.0" {
		return apperror.Field("version", "Select WFS version 1.1.0 or 2.0.0.", nil)
	}
	if err := validateWFSChoices(w.FeatureTypes, w.DefaultFeatureType, "feature_types", "default_feature_type"); err != nil {
		return err
	}
	if err := validateWFSChoices(w.OutputFormats, w.DefaultOutputFormat, "output_formats", "default_output_format"); err != nil {
		return err
	}
	if w.ResultLimit != nil && *w.ResultLimit <= 0 {
		return apperror.Field("result_limit", "Result limit must be positive.", nil)
	}
	if err := validatePersistedWebServiceParameters(w.Parameters); err != nil {
		return err
	}
	parameters, err := parseWebServiceParameterInputs(webServiceParameterInputs(w.Parameters), "parameters")
	if err != nil {
		return err
	}
	if err := validateWFSParameters(w.GetFeatureTemplateURL, parameters); err != nil {
		return err
	}
	w.Parameters, err = applyNormalizedWebServiceParameters(w.Parameters, parameters)
	return err
}

func (w *WFSWebService) NativeFormats() []NativeFormat {
	result := make([]NativeFormat, len(w.OutputFormats))
	for i, format := range w.OutputFormats {
		result[i] = NativeFormat{OutputFormat: format, Label: format}
	}
	return result
}

func (w *WFSWebService) PrepareAccess(input DistributionAccessInput) (PreparedHTTPRequest, error) {
	if w == nil {
		return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
	}
	wfs, ok := input.Input.(WFSAccessInput)
	if !ok {
		return PreparedHTTPRequest{}, apperror.Field("type", "Request type does not match service configuration.", nil)
	}
	featureType := w.DefaultFeatureType
	if wfs.FeatureType != nil {
		featureType = *wfs.FeatureType
	}
	if !slices.Contains(w.FeatureTypes, featureType) {
		return PreparedHTTPRequest{}, apperror.Field("feature_type", "Select a configured feature type.", nil)
	}
	format := w.DefaultOutputFormat
	if input.OutputFormat != nil {
		format = *input.OutputFormat
	}
	if !slices.Contains(w.OutputFormats, format) {
		return PreparedHTTPRequest{}, apperror.Field("output_format", "Select a native output format for this distribution.", nil)
	}
	limit := w.ResultLimit
	if wfs.ResultLimit != nil {
		if !w.ResultLimitEditable && (limit == nil || *wfs.ResultLimit != *limit) {
			return PreparedHTTPRequest{}, apperror.Field("result_limit", "Result limit is fixed for this distribution.", nil)
		}
		limit = wfs.ResultLimit
	}
	if limit != nil && *limit <= 0 {
		return PreparedHTTPRequest{}, apperror.Field("result_limit", "Result limit must be positive.", nil)
	}
	featureKey, limitKey := "typeName", "maxFeatures"
	switch w.Version {
	case "1.1.0":
	case "2.0.0":
		featureKey, limitKey = "typeNames", "count"
	default:
		return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
	}
	const crs = "urn:ogc:def:crs:OGC:1.3:CRS84"
	query := url.Values{
		"SERVICE":      {"WFS"},
		"REQUEST":      {"GetFeature"},
		"VERSION":      {w.Version},
		featureKey:     {featureType},
		"outputFormat": {format},
		"srsName":      {crs},
	}
	if limit != nil {
		query.Set(limitKey, strconv.FormatInt(*limit, 10))
	}
	if bbox := wfs.BBox; bbox != nil {
		coordinates := []float64{bbox.West, bbox.South, bbox.East, bbox.North}
		text := make([]string, 0, 5)
		for _, coordinate := range coordinates {
			if math.IsNaN(coordinate) || math.IsInf(coordinate, 0) {
				return PreparedHTTPRequest{}, apperror.Field("bbox", "Bounds must contain finite numbers.", nil)
			}
			text = append(text, strconv.FormatFloat(coordinate, 'g', -1, 64))
		}
		query.Set("BBOX", strings.Join(append(text, crs), ","))
	}
	renderer := GenericWebService{
		TemplateURL: w.GetFeatureTemplateURL,
		Method:      HTTPMethodGet,
		Parameters:  w.Parameters,
	}
	request, err := renderer.prepareHTTPRequest(wfs.Parameters, nil)
	if err != nil {
		return PreparedHTTPRequest{}, err
	}
	endpoint, err := url.Parse(request.URL)
	if err != nil {
		return PreparedHTTPRequest{}, apperror.Default(apperror.CodeInternal)
	}
	if endpoint.RawQuery != "" {
		endpoint.RawQuery += "&"
	}
	endpoint.RawQuery += query.Encode()
	request.URL = endpoint.String()
	return request, nil
}
