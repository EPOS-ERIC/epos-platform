import type {
  ReplaceSoftwareApplicationDraftRequest,
  ReplaceSoftwareSourceCodeDraftRequest,
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata
} from '../../generated/backofficeSchemas';

type SoftwareDetailForm =
  | SoftwareApplicationResourceMetadata
  | SoftwareSourceCodeResourceMetadata;

export function toSoftwareApplicationSaveRequest(
  form: SoftwareApplicationResourceMetadata,
  revision: number
): ReplaceSoftwareApplicationDraftRequest {
  return {
    resource_type: 'software-application',
    revision,
    metadata: {
      name: form.name || 'Untitled software application',
      description: form.description,
      keywords: form.keywords,
      license: undefinedIfBlank(form.license),
      version: undefinedIfBlank(form.version),
      download_url: undefinedIfBlank(form.download_url),
      requirements: undefinedIfBlank(form.requirements),
      operating_system: undefinedIfBlank(form.operating_system),
      main_entity_of_page: undefinedIfBlank(form.main_entity_of_page),
      creators: form.creators.filter((creator) => hasIdentifierValue(creator.id)).map((creator) => creator.id),
      identifiers: form.identifiers
        .filter((identifier) => hasIdentifierValue(identifier.type) && hasIdentifierValue(identifier.value))
        .map((identifier) => ({ type: identifier.type, value: identifier.value })),
      categories: form.categories.filter((category) => hasIdentifierValue(category.id)).map((category) => category.id),
      contact_points: (form.contact_points ?? []).filter(hasIdentifierValue)
    }
  };
}

export function toSoftwareSourceCodeSaveRequest(
  form: SoftwareSourceCodeResourceMetadata,
  revision: number
): ReplaceSoftwareSourceCodeDraftRequest {
  return {
    resource_type: 'software-source-code',
    revision,
    metadata: {
      name: form.name || 'Untitled software source code',
      description: form.description,
      keywords: form.keywords,
      repository: undefinedIfBlank(form.repository),
      programming_language: undefinedIfBlank(form.programming_language),
      license: undefinedIfBlank(form.license),
      software_requirements: undefinedIfBlank(form.software_requirements),
      runtime_platform: undefinedIfBlank(form.runtime_platform),
      main_entity_of_page: undefinedIfBlank(form.main_entity_of_page),
      version: undefinedIfBlank(form.version),
      creators: form.creators.filter((creator) => hasIdentifierValue(creator.id)).map((creator) => creator.id),
      identifiers: form.identifiers
        .filter((identifier) => hasIdentifierValue(identifier.type) && hasIdentifierValue(identifier.value))
        .map((identifier) => ({ type: identifier.type, value: identifier.value })),
      categories: form.categories.filter((category) => hasIdentifierValue(category.id)).map((category) => category.id),
      contact_points: (form.contact_points ?? []).filter(hasIdentifierValue)
    }
  };
}

export function isSoftwareDescriptionValid(form: SoftwareDetailForm): boolean {
  return Boolean(form.description.trim());
}

function undefinedIfBlank(value: string | null | undefined): string | undefined {
  if (value == null) {
    return undefined;
  }

  return hasIdentifierValue(value) ? value : undefined;
}

function hasIdentifierValue(value: string | number | null | undefined): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}
