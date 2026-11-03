import { DistributionSummary } from './distributionSummary.interface';

export interface DatasetReference {
  id: string;
  name: string;
}

export interface DatasetDistribution {
  id: string;
  name: string;
  type: 'WEB_SERVICE' | 'DOWNLOADABLE_FILE';
  webServiceType?: string;
  visibleOn?: Array<'MAP' | 'TABLE' | 'GRAPH'> | null;
  provider?: DatasetReference | null;
  summary: DistributionSummary;
}

export interface DatasetSummary {
  resourceId: string;
  versionId: string;
  name: string;
  status: string;
  updatedAt: string;
  createdBy?: DatasetReference | null;
  group: DatasetReference;
  providers: DatasetReference[];
  categories: DatasetReference[];
  distributions: DatasetDistribution[];
}
