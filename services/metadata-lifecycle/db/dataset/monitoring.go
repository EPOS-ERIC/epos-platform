package dataset

import (
	"context"
	"errors"
	"fmt"

	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/db/sqlc"
	"github.com/EPOS-ERIC/epos-platform/services/metadata-lifecycle/domain"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MonitoringDistributions loads published monitoring data in batches.
func (r *Repo) MonitoringDistributions(ctx context.Context) ([]domain.MonitoringDistribution, error) {
	rows, err := r.q.ListPublishedMonitoringDistributions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list published monitoring distributions: %w", err)
	}

	serviceIDs := make([]uuid.UUID, 0, len(rows))
	versionIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if row.Distribution.WebServiceID != nil {
			serviceIDs = append(serviceIDs, *row.Distribution.WebServiceID)
		} else {
			versionIDs = append(versionIDs, row.VersionID)
		}
	}

	services, failures, err := readMonitoringWebServices(ctx, r.q, serviceIDs)
	if err != nil {
		return nil, err
	}

	contacts, err := r.q.ListMonitoringDatasetContactPoints(ctx, versionIDs)
	if err != nil {
		return nil, fmt.Errorf("list dataset monitoring contacts: %w", err)
	}

	contactsByVersion := make(map[uuid.UUID][]domain.ContactPoint)
	for _, contact := range contacts {
		contactsByVersion[contact.DatasetVersionID] = append(contactsByVersion[contact.DatasetVersionID], toDomainContactPoint(contact.ContactPoint))
	}

	return r.mapMonitoringDistributions(ctx, rows, services, failures, contactsByVersion), nil
}

func (r *Repo) mapMonitoringDistributions(
	ctx context.Context,
	rows []sqlc.ListPublishedMonitoringDistributionsRow,
	services map[uuid.UUID]domain.WebService,
	failures map[uuid.UUID]error,
	contactsByVersion map[uuid.UUID][]domain.ContactPoint,
) []domain.MonitoringDistribution {
	result := make([]domain.MonitoringDistribution, 0, len(rows))
	for _, row := range rows {
		item := domain.MonitoringDistribution{
			ConceptID:   row.ConceptID,
			ConceptName: row.ConceptName,
			GroupID:     row.GroupID,
			GroupName:   row.GroupName,
			ResourceID:  row.ResourceID,
			VersionID:   row.VersionID,
		}

		var err error
		if row.Distribution.WebServiceID == nil && (row.Distribution.DownloadableFileUrl == nil || row.Distribution.DownloadableFileFormat == nil) {
			err = fmt.Errorf("downloadable file configuration missing")
		} else {
			item.Distribution = toDomainDistribution(row.Distribution, services)
			if row.Distribution.WebServiceID != nil {
				err = failures[*row.Distribution.WebServiceID]

				if err == nil && item.Distribution.WebService == nil {
					err = fmt.Errorf("web service missing")
				}

				if item.Distribution.WebService != nil {
					item.ContactPoints = item.Distribution.WebService.Common().ContactPoints
				}
			} else {
				item.ContactPoints = contactsByVersion[row.VersionID]
			}

			if err == nil {
				err = item.Distribution.Normalize()
			}
		}

		for i := range item.ContactPoints {
			if err != nil {
				break
			}

			if contactErr := item.ContactPoints[i].Normalize(); contactErr != nil {
				err = fmt.Errorf("invalid monitoring contact: %w", contactErr)
			}
		}

		if err != nil {
			id := row.Distribution.ID.String()
			r.log.ErrorContext(ctx, "failed to load monitoring distribution", "distribution_id", id)
			trace.SpanFromContext(ctx).RecordError(
				errors.New("failed to load monitoring distribution"),
				trace.WithAttributes(attribute.String("distribution_id", id)),
			)
			continue
		}
		result = append(result, item)
	}

	return result
}

func readMonitoringWebServices(ctx context.Context, q *sqlc.Queries, ids []uuid.UUID) (map[uuid.UUID]domain.WebService, map[uuid.UUID]error, error) {
	result := make(map[uuid.UUID]domain.WebService, len(ids))
	failures := make(map[uuid.UUID]error)
	if len(ids) == 0 {
		return result, failures, nil
	}

	rows, err := q.ListMonitoringWebServices(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring web services: %w", err)
	}

	providerIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		service, err := toDomainWebService(row)
		if err != nil {
			failures[row.ID] = err
			continue
		}

		result[row.ID] = service
		if row.ProviderOrganizationID != nil {
			providerIDs = append(providerIDs, *row.ProviderOrganizationID)
		}
	}

	generic, err := q.ListLifecycleGenericWebServices(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring generic services: %w", err)
	}

	for _, row := range generic {
		configuration := genericWebService(result[row.WebServiceID])
		if configuration == nil {
			failures[row.WebServiceID] = fmt.Errorf("invalid generic service configuration")
			continue
		}

		configuration.TemplateURL = row.TemplateUrl
		configuration.Method = domain.HTTPMethod(row.Method)
		configuration.Format = row.Format
	}

	wfs, err := q.ListLifecycleWFSServices(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring WFS services: %w", err)
	}

	for _, row := range wfs {
		service, ok := result[row.WebServiceID].(*domain.WFSWebService)
		if !ok {
			failures[row.WebServiceID] = fmt.Errorf("invalid WFS configuration")
			continue
		}

		service.GetFeatureTemplateURL = row.GetFeatureTemplateUrl
		service.Version = row.Version
		service.FeatureTypes = row.FeatureTypes
		service.DefaultFeatureType = row.DefaultFeatureType
		service.OutputFormats = row.OutputFormats
		service.DefaultOutputFormat = row.DefaultOutputFormat
		service.ResultLimit = row.ResultLimit
		service.ResultLimitEditable = row.ResultLimitEditable
	}

	wms, err := q.ListLifecycleWMSServices(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring WMS services: %w", err)
	}

	for _, row := range wms {
		service, ok := result[row.WebServiceID].(*domain.WMSWebService)
		if !ok {
			failures[row.WebServiceID] = fmt.Errorf("invalid WMS configuration")
			continue
		}

		service.CapabilitiesURL = row.CapabilitiesUrl
		service.DefaultLayer = row.DefaultLayer
		service.TransparencyEditable = row.TransparencyEditable
		service.DefaultTransparency = row.DefaultTransparency
	}

	wmsLayers, err := q.ListLifecycleWMSLayers(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring WMS layers: %w", err)
	}

	for _, row := range wmsLayers {
		service, ok := result[row.WebServiceID].(*domain.WMSWebService)
		if !ok {
			failures[row.WebServiceID] = fmt.Errorf("invalid WMS layer")
			continue
		}

		service.Layers = append(service.Layers, domain.WMSLayer{
			ID:           row.ID,
			ResourceID:   row.ResourceID,
			Name:         row.Name,
			DefaultStyle: row.DefaultStyle,
			Styles:       row.Styles,
		})
	}

	wmts, err := q.ListLifecycleWMTSServices(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring WMTS services: %w", err)
	}

	for _, row := range wmts {
		service, ok := result[row.WebServiceID].(*domain.WMTSWebService)
		if !ok {
			failures[row.WebServiceID] = fmt.Errorf("invalid WMTS configuration")
			continue
		}

		service.CapabilitiesURL = row.CapabilitiesUrl
		service.DefaultLayer = row.DefaultLayer
	}

	wmtsLayers, err := q.ListLifecycleWMTSLayers(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring WMTS layers: %w", err)
	}

	for _, row := range wmtsLayers {
		service, ok := result[row.WebServiceID].(*domain.WMTSWebService)
		if !ok {
			failures[row.WebServiceID] = fmt.Errorf("invalid WMTS layer")
			continue
		}

		service.Layers = append(service.Layers, domain.WMTSLayer{
			ID:           row.ID,
			ResourceID:   row.ResourceID,
			Name:         row.Name,
			DefaultStyle: row.DefaultStyle,
			Styles:       row.Styles,
		})
	}

	organizations, err := q.ListLifecycleOrganizations(ctx, providerIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring providers: %w", err)
	}

	providers := make(map[uuid.UUID]domain.OrganizationMetadata, len(organizations))
	for _, row := range organizations {
		providers[row.ID] = toDomainOrganizationMetadata(row)
	}

	for _, row := range rows {
		if row.ProviderOrganizationID != nil && result[row.ID] != nil {
			provider, ok := providers[*row.ProviderOrganizationID]
			if !ok {
				failures[row.ID] = fmt.Errorf("web service provider missing")
				continue
			}

			result[row.ID].Common().Provider = &provider
		}
	}

	contacts, err := q.ListLifecycleWebServiceContactPoints(ctx, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("list monitoring service contacts: %w", err)
	}

	for _, row := range contacts {
		if service := result[row.WebServiceID]; service != nil {
			service.Common().ContactPoints = append(service.Common().ContactPoints, toDomainContactPoint(row.ContactPoint))
		}
	}

	if err := readMonitoringParameters(ctx, q, ids, result, failures); err != nil {
		return nil, nil, err
	}

	return result, failures, nil
}

func readMonitoringParameters(
	ctx context.Context,
	q *sqlc.Queries,
	ids []uuid.UUID,
	services map[uuid.UUID]domain.WebService,
	failures map[uuid.UUID]error,
) error {
	bodies, err := q.ListLifecycleWebServiceBodies(ctx, ids)
	if err != nil {
		return fmt.Errorf("list monitoring request bodies: %w", err)
	}

	bodyOwners := make(map[uuid.UUID]uuid.UUID, len(bodies))
	for _, body := range bodies {
		bodyOwners[body.ID] = body.WebServiceID

		configuration := genericWebService(services[body.WebServiceID])
		if configuration == nil {
			failures[body.WebServiceID] = fmt.Errorf("invalid request body owner")
			continue
		}

		configuration.Body = &domain.RequestBody{
			Required:    body.BodyRequired,
			ContentType: body.ContentType,
			Template:    body.Template,
		}
	}

	parameters, err := q.ListLifecycleWebServiceParameters(ctx, ids)
	if err != nil {
		return fmt.Errorf("list monitoring parameters: %w", err)
	}

	templateIDs := make([]uuid.UUID, 0, len(parameters))
	parameterIDs := make([]uuid.UUID, 0, len(parameters))

	for _, parameter := range parameters {
		parameterIDs = append(parameterIDs, parameter.ID)

		if parameter.SchemaKind == sqlc.ParameterSchemaKindTEMPLATE {
			templateIDs = append(templateIDs, parameter.ID)
		}
	}

	templates, err := q.ListLifecycleParameterTemplates(ctx, templateIDs)
	if err != nil {
		return fmt.Errorf("list monitoring parameter templates: %w", err)
	}

	templatesByID := make(map[uuid.UUID]*sqlc.ParameterTemplate, len(templates))
	for i := range templates {
		templatesByID[templates[i].ParameterID] = &templates[i]
	}

	children, err := q.ListLifecycleTemplateParameters(ctx, templateIDs)
	if err != nil {
		return fmt.Errorf("list monitoring template parameters: %w", err)
	}

	childrenByParent := make(map[uuid.UUID][]sqlc.Parameter)
	for _, child := range children {
		if child.TemplateParameterID == nil {
			return fmt.Errorf("monitoring template parameter has no parent")
		}

		childrenByParent[*child.TemplateParameterID] = append(childrenByParent[*child.TemplateParameterID], child)
		parameterIDs = append(parameterIDs, child.ID)
	}

	schemas, err := readParameterSchemas(ctx, q, parameterIDs)
	if err != nil {
		return err
	}

	for _, item := range parameters {
		var serviceID uuid.UUID
		switch {
		case item.WebServiceID != nil:
			serviceID = *item.WebServiceID
		case item.WfsServiceID != nil:
			serviceID = *item.WfsServiceID
		case item.WmsGetMapServiceID != nil:
			serviceID = *item.WmsGetMapServiceID
		case item.WmsGetFeatureInfoServiceID != nil:
			serviceID = *item.WmsGetFeatureInfoServiceID
		case item.WmtsGetTileServiceID != nil:
			serviceID = *item.WmtsGetTileServiceID
		case item.WmtsGetFeatureInfoServiceID != nil:
			serviceID = *item.WmtsGetFeatureInfoServiceID
		case item.WebServiceBodyID != nil:
			serviceID = bodyOwners[*item.WebServiceBodyID]
		}

		if serviceID == uuid.Nil {
			return fmt.Errorf("monitoring parameter has no service owner")
		}

		if failures[serviceID] != nil {
			continue
		}

		parameter, err := domainParameter(item, templatesByID[item.ID], childrenByParent[item.ID], schemas)
		if err != nil {
			failures[serviceID] = fmt.Errorf("decode service parameter: %w", err)
			continue
		}

		if item.WebServiceBodyID != nil {
			configuration := genericWebService(services[serviceID])
			if configuration == nil || configuration.Body == nil {
				failures[serviceID] = fmt.Errorf("request body missing")
				continue
			}

			configuration.Body.Parameters = append(configuration.Body.Parameters, parameter)
			continue
		}

		var location domain.ParameterLocation
		if item.Location != nil {
			location = domain.ParameterLocation(*item.Location)
		}

		value := domain.WebServiceParameter{
			Parameter: parameter,
			Location:  location,
			Explode:   item.Explode,
		}

		switch service := services[serviceID].(type) {
		case *domain.HTTPWebService:
			if item.WebServiceID != nil {
				service.Parameters = append(service.Parameters, value)
			} else {
				failures[serviceID] = fmt.Errorf("invalid HTTP parameter owner")
			}
		case *domain.DynamicWebService:
			if item.WebServiceID != nil {
				service.Parameters = append(service.Parameters, value)
			} else {
				failures[serviceID] = fmt.Errorf("invalid dynamic parameter owner")
			}
		case *domain.WFSWebService:
			if item.WfsServiceID != nil {
				service.Parameters = append(service.Parameters, value)
			} else {
				failures[serviceID] = fmt.Errorf("invalid WFS parameter owner")
			}
		case *domain.WMSWebService:
			switch {
			case item.WmsGetMapServiceID != nil:
				service.GetMapParameters = append(service.GetMapParameters, value)
			case item.WmsGetFeatureInfoServiceID != nil:
				service.GetFeatureInfoParameters = append(service.GetFeatureInfoParameters, value)
			default:
				failures[serviceID] = fmt.Errorf("invalid WMS parameter owner")
			}
		case *domain.WMTSWebService:
			switch {
			case item.WmtsGetTileServiceID != nil:
				service.GetTileParameters = append(service.GetTileParameters, value)
			case item.WmtsGetFeatureInfoServiceID != nil:
				service.GetFeatureInfoParameters = append(service.GetFeatureInfoParameters, value)
			default:
				failures[serviceID] = fmt.Errorf("invalid WMTS parameter owner")
			}
		default:
			failures[serviceID] = fmt.Errorf("parameter service missing")
		}
	}

	return nil
}
