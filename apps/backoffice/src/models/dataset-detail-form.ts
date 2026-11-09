import type {
  DatasetAccrualPeriodicity,
  ParameterLocation,
  ParameterRole,
  WebServiceMethod,
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
  role?: ParameterRole | '';
};

export type DatasetWebServiceScalarParameterForm = DatasetWebServiceScalarParameterCommonForm &
  (
    | {
        kind: 'STRING';
        defaultValue: string[];
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
        defaultValue: string[];
        allowedValues: DatasetEnumOptionForm[];
      }
    | {
        kind: 'FLOAT';
        minimum: string;
        maximum: string;
        defaultValue: string[];
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
  role?: ParameterRole | '';
  explode: boolean;
};

export type DatasetWebServiceParameterForm = DatasetWebServiceParameterCommonForm &
  (
    | {
        kind: 'STRING';
        multipleValues: boolean;
        defaultValue: string[];
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
        defaultValue: string[];
        allowedValues: DatasetEnumOptionForm[];
      }
    | {
        kind: 'FLOAT';
        multipleValues: boolean;
        minimum: string;
        maximum: string;
        defaultValue: string[];
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

export type DatasetHttpWebServiceConfigurationForm = {
  type: 'HTTP' | 'DYNAMIC';
  templateUrl: string;
  method: WebServiceMethod;
  format: string;
  bodyContentType: string;
  bodyTemplate: string;
  bodyRequired: boolean;
  bodyParameters: DatasetWebServiceScalarParameterForm[];
  parameters: DatasetWebServiceParameterForm[];
};

export type DatasetWfsWebServiceConfigurationForm = {
  type: 'WFS';
  getFeatureTemplateUrl: string;
  version: '1.1.0' | '2.0.0';
  featureTypes: string[];
  defaultFeatureType: string;
  outputFormats: string[];
  defaultOutputFormat: string;
  resultLimit: string;
  resultLimitEditable: boolean;
  parameters: DatasetWebServiceParameterForm[];
};

export type DatasetWmsLayerForm = {
  name: string;
  defaultStyle: string;
  styles: string[];
};

export type DatasetWmsWebServiceConfigurationForm = {
  type: 'WMS';
  capabilitiesUrl: string;
  defaultLayer: string;
  layers: DatasetWmsLayerForm[];
  transparencyEditable: boolean;
  defaultTransparency: boolean;
  getMapParameters: DatasetWebServiceParameterForm[];
  getFeatureInfoParameters: DatasetWebServiceParameterForm[];
};

export type DatasetWmtsLayerForm = {
  name: string;
  defaultStyle: string;
  styles: string[];
};

export type DatasetWmtsWebServiceConfigurationForm = {
  type: 'WMTS';
  capabilitiesUrl: string;
  defaultLayer: string;
  layers: DatasetWmtsLayerForm[];
  getTileParameters: DatasetWebServiceParameterForm[];
  getFeatureInfoParameters: DatasetWebServiceParameterForm[];
};

export type DatasetWebServiceConfigurationForm =
  | DatasetHttpWebServiceConfigurationForm
  | DatasetWfsWebServiceConfigurationForm
  | DatasetWmsWebServiceConfigurationForm
  | DatasetWmtsWebServiceConfigurationForm;

export type DatasetWebServiceForm = {
  visibleOn: WebServiceVisibleOn[];
  documentationUri: string;
  provider: DatasetWebServiceProviderForm;
  contactPoints: DatasetWebServiceContactPointForm[];
  configuration: DatasetWebServiceConfigurationForm;
};

export type DatasetDistributionCommonForm = {
  id?: string;
  nativeFormats?: string[];
  title: string;
  description: string;
  license: string;
  byteSize: string;
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
  accrualPeriodicity: Exclude<DatasetAccrualPeriodicity, null> | '';
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
