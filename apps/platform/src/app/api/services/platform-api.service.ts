import { HttpClient } from '@angular/common/http';
import { Inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { API_BASE_URL } from '../config/api-base-url.token';
import {
  ConceptResponse,
  ConceptSchemeResponse,
  ConceptTreeResponse,
  DatasetResponse,
  DistributionResponse,
  OrganizationResponse,
  ResourceId,
  SearchQueryParams,
  SearchResponse,
  SoftwareApplicationResponse,
  SoftwareSourceCodeResponse,
  WebserviceResponse,
} from '../types/api-types';

@Injectable({ providedIn: 'root' })
export class PlatformApiService {
  constructor(
    private readonly httpClient: HttpClient,
    @Inject(API_BASE_URL) private readonly apiBaseUrl: string,
  ) {}

  public search(params: SearchQueryParams): Observable<SearchResponse> {
    return this.httpClient.get<SearchResponse>(this.buildUrl('/search'), { params });
  }

  public getDatasetById(id: ResourceId): Observable<DatasetResponse> {
    return this.httpClient.get<DatasetResponse>(this.buildUrl(`/dataset/${id}`));
  }

  public getDistributionById(id: ResourceId): Observable<DistributionResponse> {
    return this.httpClient.get<DistributionResponse>(this.buildUrl(`/distribution/${id}`));
  }

  public getSoftwareApplicationById(id: ResourceId): Observable<SoftwareApplicationResponse> {
    return this.httpClient.get<SoftwareApplicationResponse>(
      this.buildUrl(`/softwareapplication/${id}`),
    );
  }

  public getSoftwareSourceCodeById(id: ResourceId): Observable<SoftwareSourceCodeResponse> {
    return this.httpClient.get<SoftwareSourceCodeResponse>(
      this.buildUrl(`/softwaresourcecode/${id}`),
    );
  }

  public getWebserviceById(id: ResourceId): Observable<WebserviceResponse> {
    return this.httpClient.get<WebserviceResponse>(this.buildUrl(`/webservice/${id}`));
  }

  public getOrganizationById(id: ResourceId): Observable<OrganizationResponse> {
    return this.httpClient.get<OrganizationResponse>(this.buildUrl(`/organization/${id}`));
  }

  public getConceptTree(): Observable<ConceptTreeResponse> {
    return this.httpClient.get<ConceptTreeResponse>(this.buildUrl('/concept'));
  }

  public getConceptById(id: ResourceId): Observable<ConceptResponse> {
    return this.httpClient.get<ConceptResponse>(this.buildUrl(`/concept/${id}`));
  }

  public getConceptSchemeById(id: ResourceId): Observable<ConceptSchemeResponse> {
    return this.httpClient.get<ConceptSchemeResponse>(this.buildUrl(`/conceptscheme/${id}`));
  }

  public getByHref(href: string): Observable<unknown> {
    return this.httpClient.get<unknown>(this.resolveHref(href));
  }

  private buildUrl(path: string): string {
    return `${this.apiBaseUrl}${path}`;
  }

  private resolveHref(href: string): string {
    if (href.startsWith('http://') || href.startsWith('https://')) {
      return href;
    }

    return this.buildUrl(href.startsWith('/') ? href : `/${href}`);
  }
}
