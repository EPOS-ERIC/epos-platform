import type {
  Documentation,
  DownloadableFile,
  ParameterMetadata,
  ResourceVersionReference,
  TemporalCoverage
} from '../../generated/backofficeSchemas';

export type DatasetSpatialForm = string | null;

export type DatasetTemporalForm = TemporalCoverage;

export type DatasetPersistentIdentifierForm = {
  id?: string;
  type: string;
  value: string;
};

export type DatasetProviderForm = ResourceVersionReference;

export type DatasetCategoryForm = ResourceVersionReference;

export type DatasetWebServiceDocumentationForm = Documentation;

export type DatasetWebServiceContactPointForm = string;

export type DatasetWebServiceProviderForm = ResourceVersionReference;

export type DatasetWebServiceParameterMetadataForm = ParameterMetadata;

export type DatasetWebServiceParameterForm =
  | {
      kind: 'STRING';
      metadata: DatasetWebServiceParameterMetadataForm;
      defaultValue: string | null;
      allowedValues: string[];
      pattern: string;
    }
  | {
      kind: 'BOOLEAN';
      metadata: DatasetWebServiceParameterMetadataForm;
      defaultValue: boolean | null;
    }
  | {
      kind: 'INTEGER';
      metadata: DatasetWebServiceParameterMetadataForm;
      minimum: string;
      maximum: string;
      defaultValue: string;
      allowedValues: string[];
    }
  | {
      kind: 'FLOAT';
      metadata: DatasetWebServiceParameterMetadataForm;
      minimum: string;
      maximum: string;
      defaultValue: string;
      allowedValues: string[];
    }
  | {
      kind: 'DATETIME';
      metadata: DatasetWebServiceParameterMetadataForm;
      minimum: string;
      maximum: string;
      defaultValue: string;
      pattern: string;
    };

export type DatasetWebServiceForm = {
  name: string;
  description: string;
  templateUrl: string;
  format: string;
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
  type: string;
};

export type DatasetDownloadableFileForm = DownloadableFile;

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
  keywords: string;
  accrualPeriodicity: string;
  qualityAnnotation: string;
  created: string;
  updated: string;
  spatial: DatasetSpatialForm;
  temporal: DatasetTemporalForm;
  persistentIdentifiers: DatasetPersistentIdentifierForm[];
  contactPoints: DatasetWebServiceContactPointForm[];
  providers: DatasetProviderForm[];
  categories: DatasetCategoryForm[];
  distributions: DatasetDistributionForm[];
};
