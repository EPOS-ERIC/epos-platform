import type {
  ParameterLocation,
  WebServiceMethod,
  WebServiceType,
  WebServiceVisibleOn,
} from '../../generated/backofficeSchemas';

export type DatasetSpatialForm = string | null;

export type DatasetTemporalForm = {
  start: string;
  end: string;
};

export type DatasetGeologicalTemporalForm = {
  startMya: string;
  endMya: string;
};

export type DatasetPersistentIdentifierForm = {
  id?: string;
  type: string;
  value: string;
};

export type DatasetProviderForm = {
  id: string;
  name: string;
};

export type DatasetCategoryForm = DatasetProviderForm;

export type DatasetWebServiceDocumentationForm = {
  title: string;
  description: string;
  uri: string;
};

export type DatasetWebServiceContactPointForm = string;

export type DatasetWebServiceProviderForm = DatasetProviderForm;

export type DatasetEnumOptionForm = {
  value: string;
  label: string;
};

export type DatasetWebServiceScalarParameterCommonForm = {
  position: number;
  name: string;
  label: string;
  description: string;
  required: boolean;
  readOnly: boolean;
};

export type DatasetWebServiceScalarParameterForm = DatasetWebServiceScalarParameterCommonForm &
  (
    | {
        kind: 'STRING';
        defaultValue: string | null;
        allowedValues: DatasetEnumOptionForm[];
        pattern: string;
      }
    | {
        kind: 'BOOLEAN';
        defaultValue: boolean | null;
      }
    | {
        kind: 'INTEGER';
        minimum: string;
        maximum: string;
        defaultValue: string;
        allowedValues: DatasetEnumOptionForm[];
      }
    | {
        kind: 'FLOAT';
        minimum: string;
        maximum: string;
        defaultValue: string;
        allowedValues: DatasetEnumOptionForm[];
      }
    | {
        kind: 'DATETIME';
        minimum: string;
        maximum: string;
        defaultValue: string;
        layout: string;
      }
  );

export type DatasetWebServiceParameterCommonForm = DatasetWebServiceScalarParameterCommonForm & {
  id: string;
  property: string;
  location: ParameterLocation;
  explode: boolean;
};

export type DatasetWebServiceParameterForm = DatasetWebServiceParameterCommonForm &
  (
    | {
        kind: 'STRING';
        multipleValues: boolean;
        defaultValue: string | null;
        allowedValues: DatasetEnumOptionForm[];
        pattern: string;
      }
    | {
        kind: 'BOOLEAN';
        defaultValue: boolean | null;
      }
    | {
        kind: 'INTEGER';
        multipleValues: boolean;
        minimum: string;
        maximum: string;
        defaultValue: string;
        allowedValues: DatasetEnumOptionForm[];
      }
    | {
        kind: 'FLOAT';
        multipleValues: boolean;
        minimum: string;
        maximum: string;
        defaultValue: string;
        allowedValues: DatasetEnumOptionForm[];
      }
    | {
        kind: 'DATETIME';
        minimum: string;
        maximum: string;
        defaultValue: string;
        layout: string;
      }
    | {
        kind: 'TEMPLATE';
        template: string;
        parameters: DatasetWebServiceScalarParameterForm[];
      }
  );

export type DatasetWebServiceForm = {
  name: string;
  description: string;
  templateUrl: string;
  method: WebServiceMethod;
  type: WebServiceType;
  visibleOn: WebServiceVisibleOn[];
  format: string;
  bodyContentType: string;
  bodyTemplate: string;
  bodyParameters: DatasetWebServiceScalarParameterForm[];
  documentation: DatasetWebServiceDocumentationForm;
  provider: DatasetWebServiceProviderForm;
  contactPoints: DatasetWebServiceContactPointForm[];
  spatialCoverage: DatasetSpatialForm;
  temporalCoverage: DatasetTemporalForm;
  parameters: DatasetWebServiceParameterForm[];
};

export type DatasetDistributionCommonForm = {
  title: string;
  description: string;
  issued: string;
  modified: string;
  license: string;
  type: 'WEB_SERVICE' | 'DOWNLOADABLE_FILE';
};

export type DatasetDownloadableFileForm = {
  url: string;
  format: string;
};

export type DatasetDistributionForm = DatasetDistributionCommonForm &
  (
    | {
        kind: 'web-service';
        webService: DatasetWebServiceForm;
      }
    | {
        kind: 'downloadable-file';
        downloadableFile: DatasetDownloadableFileForm;
      }
  );

export type DatasetDetailForm = {
  title: string;
  description: string;
  keywords: string[];
  accrualPeriodicity: string;
  qualityAnnotation: string;
  created: string;
  updated: string;
  spatial: DatasetSpatialForm;
  temporalMode: 'calendar' | 'geological';
  temporal: DatasetTemporalForm;
  geologicalTemporal: DatasetGeologicalTemporalForm;
  persistentIdentifiers: DatasetPersistentIdentifierForm[];
  contactPoints: DatasetWebServiceContactPointForm[];
  providers: DatasetProviderForm[];
  categories: DatasetCategoryForm[];
  distributions: DatasetDistributionForm[];
};
