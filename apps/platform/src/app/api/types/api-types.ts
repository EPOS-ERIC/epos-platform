import { components, operations } from '../generated/schema';

export type ResourceId = components['parameters']['ResourceId'];
export type SearchQueryParams = NonNullable<operations['searchMetadata']['parameters']['query']>;
export type SearchResponse =
  operations['searchMetadata']['responses'][200]['content']['application/json'];
export type SearchItem = components['schemas']['SearchItem'];

export type DatasetResponse = components['schemas']['Dataset'];
export type DistributionResponse = components['schemas']['Distribution'];
export type SoftwareApplicationResponse = components['schemas']['SoftwareApplication'];
export type SoftwareSourceCodeResponse = components['schemas']['SoftwareSourceCode'];
export type WebserviceResponse = components['schemas']['Webservice'];
export type OrganizationResponse = components['schemas']['Organization'];
export type ConceptTreeResponse =
  operations['getConceptTree']['responses'][200]['content']['application/json'];
export type ConceptResponse = components['schemas']['Concept'];
export type ConceptSchemeResponse = components['schemas']['ConceptScheme'];
