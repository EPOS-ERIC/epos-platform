import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import type {
  CanonicalOrganization,
  Category,
  ContactPoint,
  CreateCategoryRequest,
  CreateContactPointRequest,
  CreateMembershipRequest,
  CreateRoleRequest,
  CreateOrganizationRequest,
  CreateResourceDraftRequest,
  DatasetResourceMetadata,
  DiscoveryItem,
  DiscoveryResourceType,
  Group,
  GroupMembership,
  GroupNotificationTopicKey,
  GroupPermission,
  GlobalNotificationTopicKey,
  NotificationSubscription,
  NotificationTopic,
  ResourceVersionReference,
  ResourceVersionSummary,
  RoleRequest,
  ResourceVersion,
  RejectResourceRequest,
  ReplaceCategoryRequest,
  ReplaceContactPointRequest,
  ReplaceOrganizationRequest,
  WriteResult,
  DatasetDraftMetadata,
  ReplaceResourceDraftRequest,
  SoftwareApplicationDraftMetadata,
  SoftwareSourceCodeDraftMetadata,
  SoftwareApplicationResourceMetadata,
  SoftwareSourceCodeResourceMetadata,
  User,
  UserListItem,
  MembershipRole,
  Permission
} from '../../generated/backofficeSchemas';
import type {
  DateFrom,
  DateTo,
  Decision,
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
    return this.http.get<ResourceVersionResponse<T>>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}`,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  listResourceVersions(
    resourceType: DiscoveryResourceType,
    id: string
  ): Observable<ResourceVersionSummary[]> {
    return this.http.get<ResourceVersionSummary[]>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions`,
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

  createContactPoint(payload: CreateContactPointRequest): Observable<ContactPoint> {
    return this.http.post<ContactPoint>(`${this.baseUrl}/contact-points`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  replaceContactPoint(id: string, payload: ReplaceContactPointRequest): Observable<ContactPoint> {
    return this.http.put<ContactPoint>(`${this.baseUrl}/contact-points/${encodeURIComponent(id)}`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  deleteContactPoint(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/contact-points/${encodeURIComponent(id)}`, {
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

  getCanonicalOrganizations(): Observable<CanonicalOrganization[]> {
    return this.http.get<CanonicalOrganization[]>(`${this.baseUrl}/organizations`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  createOrganization(payload: CreateOrganizationRequest): Observable<CanonicalOrganization> {
    return this.http.post<CanonicalOrganization>(`${this.baseUrl}/organizations`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  replaceOrganization(id: string, payload: ReplaceOrganizationRequest): Observable<CanonicalOrganization> {
    return this.http.put<CanonicalOrganization>(`${this.baseUrl}/organizations/${encodeURIComponent(id)}`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  deleteOrganization(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/organizations/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  getGroupCategories(groupId: string): Observable<Category[]> {
    return this.http.get<Category[]>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/categories`);
  }

  createGroupCategory(groupId: string, payload: CreateCategoryRequest): Observable<Category> {
    return this.http.post<Category>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/categories`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  replaceGroupCategory(groupId: string, categoryId: string, payload: ReplaceCategoryRequest): Observable<Category> {
    return this.http.put<Category>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/categories/${encodeURIComponent(categoryId)}`,
      payload,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  deleteGroupCategory(groupId: string, categoryId: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/categories/${encodeURIComponent(categoryId)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  getGroups(): Observable<Group[]> {
    return this.http.get<Group[]>(`${this.baseUrl}/groups`);
  }

  listPermissions(): Observable<Permission[]> {
    return this.http.get<Permission[]>(`${this.baseUrl}/permissions`);
  }

  listNotificationTopics(): Observable<NotificationTopic[]> {
    return this.http.get<NotificationTopic[]>(`${this.baseUrl}/notification-topics`);
  }

  listUserNotificationSubscriptions(userId: string): Observable<NotificationSubscription[]> {
    return this.http.get<NotificationSubscription[]>(
      `${this.baseUrl}/users/${encodeURIComponent(userId)}/notification-subscriptions`,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  setUserNotificationSubscription(userId: string, topicKey: GlobalNotificationTopicKey, enabled: boolean): Observable<void> {
    const url = `${this.baseUrl}/users/${encodeURIComponent(userId)}/notification-subscriptions/${encodeURIComponent(topicKey)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  setGroupUserNotificationSubscription(
    groupId: string,
    userId: string,
    topicKey: GroupNotificationTopicKey,
    enabled: boolean
  ): Observable<void> {
    const url = `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/notification-subscriptions/${encodeURIComponent(userId)}/${encodeURIComponent(topicKey)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  getGroupMemberships(groupId: string): Observable<GroupMembership[]> {
    return this.http.get<GroupMembership[]>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  requestGroupMembership(groupId: string, payload: CreateMembershipRequest): Observable<GroupMembership> {
    return this.http.post<GroupMembership>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/membership-requests`,
      payload,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  withdrawGroupMembershipRequest(groupId: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/membership-requests`, {
      headers: { Authorization: `Bearer ${this.bearerToken}` }
    });
  }

  requestGroupRole(groupId: string, payload: CreateRoleRequest): Observable<RoleRequest> {
    return this.http.post<RoleRequest>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/role-requests`,
      payload,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  leaveOwnGroupMembership(groupId: string): Observable<void> {
    return this.http.delete<void>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/self`,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  removeGroupMember(groupId: string, userId: string): Observable<void> {
    return this.http.delete<void>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/${encodeURIComponent(userId)}`,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  decideGroupMembership(groupId: string, userId: string, decision: Decision): Observable<GroupMembership> {
    return this.http.post<GroupMembership>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/${encodeURIComponent(userId)}/${decision}`,
      undefined,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  decideGroupRoleRequest(groupId: string, requestId: string, decision: Decision): Observable<GroupMembership> {
    return this.http.post<GroupMembership>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/role-requests/${encodeURIComponent(requestId)}/${decision}`,
      undefined,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
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

}
