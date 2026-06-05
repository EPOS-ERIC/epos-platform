import { components, operations } from '../generated/schema';

export type ResourceId = components['parameters']['ResourceId'];
export type SearchQueryParams = NonNullable<operations['searchMetadata']['parameters']['query']>;
export type SearchResponse = operations['searchMetadata']['responses'][200]['content']['application/json'];
export type SearchItem = components['schemas']['SearchItem'];

export type DatasetResponse = operations['getDatasetById']['responses'][200]['content']['application/json'];
export type DistributionResponse = operations['getDistributionById']['responses'][200]['content']['application/json'];
export type SoftwareApplicationResponse = operations['getSoftwareApplicationById']['responses'][200]['content']['application/json'];
export type SoftwareSourceCodeResponse = operations['getSoftwareSourceCodeById']['responses'][200]['content']['application/json'];
export type WebserviceResponse = operations['getWebserviceById']['responses'][200]['content']['application/json'];
export type OrganizationResponse = operations['getOrganizationById']['responses'][200]['content']['application/json'];
export type ConceptTreeResponse = operations['getConceptTree']['responses'][200]['content']['application/json'];
export type ConceptResponse = operations['getConceptById']['responses'][200]['content']['application/json'];
export type ConceptSchemeResponse = operations['getConceptSchemeById']['responses'][200]['content']['application/json'];
