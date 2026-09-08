import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable, of } from 'rxjs';

import type {
  CanonicalOrganization,
  Category,
  ContactPoint,
  CreateResourceDraftRequest,
  DatasetResourceMetadata,
  DiscoveryItem,
  DiscoveryResourceType,
  Group,
  ResourceVersionReference,
  ResourceVersion,
  RejectResourceRequest,
  WriteResult,
  DatasetDraftMetadata,
  ReplaceResourceDraftRequest,
  SoftwareApplicationDraftMetadata,
  SoftwareSourceCodeDraftMetadata,
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata,
  User,
  UserListItem,
  MembershipRole
} from '../../generated/backofficeSchemas';
import type {
  DateFrom,
  DateTo,
  GroupPermission,
  GroupFilter,
  Keywords,
  Provider,
  Q,
  SpatialFilter,
  TypeFilter,
  VersioningStatus
} from '../../generated/backofficeParameters';
import { AuthService } from './auth.service';

export interface DiscoverySearchParams {
  q?: Q;
  groupId?: GroupFilter;
  type?: TypeFilter;
  spatialFilter?: SpatialFilter;
  dateFrom?: DateFrom;
  dateTo?: DateTo;
  provider?: Provider;
  versioningStatus?: VersioningStatus;
  keywords?: Keywords;
}

type ResourceVersionResponseMap = {
  dataset: DatasetResourceMetadata;
  'software-application': SoftwareApplicationResourceMetadata;
  'software-source-code': SoftwareSourceCodeResourceMetadata;
};

export type ResourceVersionResponse<T extends DiscoveryResourceType> = Omit<ResourceVersion, 'metadata'> & {
  metadata: ResourceVersionResponseMap[T];
};

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private readonly http = inject(HttpClient);
  private readonly authService = inject(AuthService);
  private readonly baseUrl = '/api/v1';
  private get bearerToken(): string {
    return this.authService.accessToken;
  }
  getDiscovery(params: DiscoverySearchParams = {}): Observable<DiscoveryItem[]> {
    return this.http.get<DiscoveryItem[]>(`${this.baseUrl}/search`, {
      params: this.toHttpParams(params),
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  getResourceVersion<T extends DiscoveryResourceType>(resourceType: T, id: string, versionId: string): Observable<ResourceVersionResponse<T>> {
    if (resourceType === 'dataset') {
      const demoVersion = this.demoResourceVersions[`${id}:${versionId}`];

      if (demoVersion) {
        return of(demoVersion as ResourceVersionResponse<T>);
      }
    }

    return this.http.get<ResourceVersionResponse<T>>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}`,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  saveResourceVersionDraft(
    id: string,
    versionId: string,
    targetStatus: 'draft' | 'submitted',
    payload: ReplaceResourceDraftRequest
  ): Observable<ResourceVersion> {
    console.warn('Saving resource version payload', payload);

    return this.http.put<ResourceVersion>(
      `${this.baseUrl}/resources/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/${targetStatus}`,
      payload,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  deleteResourceVersionDraft(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<void> {
    return this.http.delete<void>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/draft`,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  submitResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/submit`,
      undefined,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  publishResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/publish`,
      undefined,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  createDraftFromPublication(resourceType: DiscoveryResourceType, id: string): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/draft-from-publication`,
      undefined,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  createDraftFromArchive(resourceType: DiscoveryResourceType, id: string): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/draft-from-archive`,
      undefined,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  unpublishResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/unpublish`,
      undefined,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  reopenResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/reopen`,
      undefined,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  rejectResourceVersion(
    resourceType: DiscoveryResourceType,
    id: string,
    versionId: string,
    payload: RejectResourceRequest
  ): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/reject`,
      payload,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  private readonly demoResourceVersions: Record<string, ResourceVersionResponse<'dataset'>> = {
    '00000000-0000-4000-8000-000000000001:10000000-0000-4000-8000-000000000001': this.buildDemoDatasetVersion('PUBLISHED', '00000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', 'Published'),
    '00000000-0000-4000-8000-000000000002:10000000-0000-4000-8000-000000000002': this.buildDemoDatasetVersion('SUBMITTED', '00000000-0000-4000-8000-000000000002', '10000000-0000-4000-8000-000000000002', 'Submitted'),
    '00000000-0000-4000-8000-000000000003:10000000-0000-4000-8000-000000000003': this.buildDemoDatasetVersion('ARCHIVED', '00000000-0000-4000-8000-000000000003', '10000000-0000-4000-8000-000000000003', 'Archived')
  };

  getContactPoint(id: string): Observable<ContactPoint> {
    return this.http.get<ContactPoint>(`${this.baseUrl}/contact-points/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  getContactPoints(): Observable<ContactPoint[]> {
    return this.http.get<ContactPoint[]>(`${this.baseUrl}/contact-points`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  getOrganizations(): Observable<ResourceVersionReference[]> {
    return this.http.get<CanonicalOrganization[]>(`${this.baseUrl}/organizations`).pipe(
      map((organizations) => {
        const references = new Map<string, ResourceVersionReference>();

        for (const { id, name } of organizations) {
          if (id && !references.has(id)) {
            references.set(id, { id, name });
          }
        }

        return Array.from(references.values());
      })
    );
  }

  getGroupCategories(groupId: string): Observable<Category[]> {
    return this.http.get<Category[]>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/categories`);
  }

  getGroups(): Observable<Group[]> {
    return this.http.get<Group[]>(`${this.baseUrl}/groups`);
  }

  getUsers(): Observable<UserListItem[]> {
    return this.http.get<UserListItem[]>(`${this.baseUrl}/users`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  getUser(id: string): Observable<User> {
    return this.http.get<User>(`${this.baseUrl}/users/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  setGroupMembershipRole(groupId: string, userId: string, role: MembershipRole, enabled: boolean): Observable<void> {
    const url = `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/${encodeURIComponent(userId)}/roles/${encodeURIComponent(role)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  setGroupMembershipPermission(groupId: string, userId: string, permission: GroupPermission, enabled: boolean): Observable<void> {
    const url = `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/${encodeURIComponent(userId)}/permissions/${encodeURIComponent(permission)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  createResourceDraft(groupId: string, payload: CreateResourceDraftRequest): Observable<WriteResult> {
    return this.http.post<WriteResult>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/resources`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  private toHttpParams(params: DiscoverySearchParams): HttpParams {
    let httpParams = new HttpParams();

    if (params.q) {
      httpParams = httpParams.set('q', params.q);
    }

    if (params.groupId) {
      httpParams = httpParams.set('groupId', params.groupId);
    }

    if (params.type?.length) {
      httpParams = httpParams.set('type', params.type.join(','));
    }

    if (params.spatialFilter) {
      httpParams = httpParams.set('spatialFilter', params.spatialFilter);
    }

    if (params.dateFrom) {
      httpParams = httpParams.set('dateFrom', params.dateFrom);
    }

    if (params.dateTo) {
      httpParams = httpParams.set('dateTo', params.dateTo);
    }

    if (params.provider?.length) {
      httpParams = httpParams.set('provider', params.provider.join(','));
    }

    if (params.versioningStatus?.length) {
      httpParams = httpParams.set('versioningStatus', params.versioningStatus.join(','));
    }

    if (params.keywords?.length) {
      httpParams = httpParams.set('keywords', params.keywords.join(','));
    }

    return httpParams;
  }

  private buildDemoDatasetVersion(
    status: 'PUBLISHED' | 'SUBMITTED' | 'ARCHIVED',
    resourceId: string,
    versionId: string,
    label: string
  ): ResourceVersionResponse<'dataset'> {
    return {
      resource_id: resourceId,
      version_id: versionId,
      revision: 1,
      status,
      created_by: 'Admin Admin',
      group_id: '23b9b58a-9f1c-5d18-85f9-73842733835c',
      group_name: 'Seismology',
      created_at: '2026-08-28T08:13:51.275437Z',
      updated_at: '2026-08-28T09:13:51.275437Z',
      metadata: {
        resource_type: 'dataset',
        id: resourceId,
        title: `TEST1 - Historical earthquakes 1000-1899 (${label})`,
        description: 'Dataset used to test detail page editability by lifecycle status.',
        keywords: ['earthquakes', 'historical', 'demo'],
        accrual_periodicity: 'IRREGULAR',
        quality_annotation: 'Demo resource for UI testing.',
        access_right: 'PUBLIC',
        distributions: [
          {
            id: '1515b3f1-3516-4442-9ca9-fbe01e723ff5',
            title: 'TEST1 - Parameters of historical earthquakes (1000-1899) - FDSN event',
            description: 'Demo web service distribution.',
            license: 'CC-BY-4.0',
            type: 'http://publications.europa.eu/resource/authority/distribution-type/WEB_SERVICE',
            issued: '2026-08-28T08:13:51.275437Z',
            modified: '2026-08-28T09:13:51.275437Z',
            byte_size: 1,
            web_service: {
              id: '11111111-1111-4111-8111-111111111111',
              name: 'FDSN event',
              description: 'Demo web service for lifecycle tests.',
              template_url: 'https://example.org/fdsn?start={start}&end={end}',
              format: 'json',
              documentation: {
                title: 'Service documentation',
                description: 'Demo documentation.',
                uri: 'https://example.org/docs'
              },
              provider: {
                id: '94f4885e-502e-5f33-ba82-033a72efd827',
                name: 'INGV - Istituto Nazionale di Geofisica e Vulcanologia'
              },
              contact_points: [
                'c8f1d1a0-1111-4444-8888-111111111111',
                'c8f1d1a0-2222-4444-8888-222222222222'
              ],
              spatial_coverage: 'POLYGON ((12.5 41.9, 12.7 42.1, 13 42, 12.9 41.8, 12.5 41.9))',
              temporal_coverage: {
                start: '2026-08-28T00:00:00Z',
                end: '2026-08-29T00:00:00Z'
              },
              parameters: []
            }
          }
        ],
        spatial_coverage: 'POINT (12.4964 41.9028)',
        temporal_coverage: {
          start: '2026-08-28T00:00:00Z',
          end: '2026-08-29T00:00:00Z'
        },
        persistent_identifiers: [
          {
            id: '11111111-1111-4111-8111-111111111111',
            type: 'DOI',
            value: '10.0000/demo.earthquake'
          }
        ],
        contact_points: ['c8f1d1a0-1111-4444-8888-111111111111', 'c8f1d1a0-2222-4444-8888-222222222222'],
        providers: [
          {
            id: '94f4885e-502e-5f33-ba82-033a72efd827',
            name: 'INGV - Istituto Nazionale di Geofisica e Vulcanologia'
          }
        ],
        categories: [
          {
            id: '53e3ec5c-9ae8-5ff5-9c35-50011b8a3921',
            name: 'Earthquake parameters'
          }
        ],
        created: '2026-08-28T08:13:51.275437Z',
        updated: '2026-08-28T09:13:51.275437Z'
      }
    };
  }
}
