import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import type {
  BlockUserRequest,
  CanonicalOrganization,
  Category,
  ContactPoint,
  Concept,
  ConversionRelation,
  CreateCategoryRequest,
  CreateContactPointRequest,
  CreateConversionPluginRequest,
  CreateGroupRequest,
  CreateMembershipRequest,
  CreateRoleRequest,
  CreateOrganizationRequest,
  CreateResourceDraftRequest,
  DatasetResourceMetadata,
  DistributionConversionPlugin,
  DiscoveryItem,
  DiscoveryResourceType,
  Group,
  GroupMembership,
  GroupNotificationTopicKey,
  GroupPermission,
  GlobalPermission,
  ManagedConversionPlugin,
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
  ReplaceConversionPluginRequest,
  ReplaceGroupRequest,
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
  WebServiceVisibleOn,
  MembershipRole,
  Permission
} from '../../generated/backofficeSchemas';
import type {
  DateFrom,
  DateTo,
  Decision,
  EditorFilter,
  GlobalRole,
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
  editorId?: EditorFilter;
  groupId?: GroupFilter;
  type?: TypeFilter;
  spatialFilter?: SpatialFilter;
  dateFrom?: DateFrom;
  dateTo?: DateTo;
  provider?: Provider;
  versioningStatus?: VersioningStatus;
  keywords?: Keywords;
}

export interface DistributionConversionRequest {
  input_format: string;
  visualizations: WebServiceVisibleOn[];
  message: string;
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
  private readonly baseUrl = 'api/v1';
  private get bearerToken(): string {
    return this.authService.accessToken;
  }
  public getDiscovery(params: DiscoverySearchParams = {}): Observable<DiscoveryItem[]> {
    return this.http.get<DiscoveryItem[]>(`${this.baseUrl}/search`, {
      params: this.toHttpParams(params),
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public getResourceVersion<T extends DiscoveryResourceType>(resourceType: T, id: string, versionId: string): Observable<ResourceVersionResponse<T>> {
    return this.http.get<ResourceVersionResponse<T>>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}`,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  public listDistributionConversionRelations(distributionId: string): Observable<DistributionConversionPlugin[]> {
    return this.http.get<DistributionConversionPlugin[]>(
      `${this.baseUrl}/distributions/${encodeURIComponent(distributionId)}/conversion-relations`,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  public listConversionPlugins(): Observable<ManagedConversionPlugin[]> {
    return this.http.get<ManagedConversionPlugin[]>(`${this.baseUrl}/conversion-plugins`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public listConversionRelations(): Observable<ConversionRelation[]> {
    return this.http.get<ConversionRelation[]>(`${this.baseUrl}/conversion-relations`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public createConversionPlugin(payload: CreateConversionPluginRequest): Observable<ManagedConversionPlugin> {
    return this.http.post<ManagedConversionPlugin>(`${this.baseUrl}/conversion-plugins`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public replaceConversionPlugin(id: string, payload: ReplaceConversionPluginRequest): Observable<ManagedConversionPlugin> {
    return this.http.put<ManagedConversionPlugin>(`${this.baseUrl}/conversion-plugins/${encodeURIComponent(id)}`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public syncConversionPlugin(id: string): Observable<ManagedConversionPlugin> {
    return this.http.post<ManagedConversionPlugin>(`${this.baseUrl}/conversion-plugins/${encodeURIComponent(id)}/sync`, null, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public deleteConversionPlugin(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/conversion-plugins/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public requestDistributionConversion(
    distributionId: string,
    payload: DistributionConversionRequest
  ): Observable<void> {
    return this.http.post<void>(
      `${this.baseUrl}/distributions/${encodeURIComponent(distributionId)}/conversion-requests`,
      payload,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  public listResourceVersions(
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

  public saveResourceVersionDraft(
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

  public deleteResourceVersionDraft(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<void> {
    return this.http.delete<void>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/draft`,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  public submitResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
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

  public withdrawResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(id)}/versions/${encodeURIComponent(versionId)}/withdraw`,
      undefined,
      {
        headers: {
          Authorization: `Bearer ${this.bearerToken}`
        }
      }
    );
  }

  public publishResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
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

  public createDraftFromPublication(resourceType: DiscoveryResourceType, id: string): Observable<WriteResult> {
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

  public createDraftFromArchive(resourceType: DiscoveryResourceType, id: string): Observable<WriteResult> {
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

  public unpublishResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
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

  public reopenResourceVersion(resourceType: DiscoveryResourceType, id: string, versionId: string): Observable<WriteResult> {
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

  public rejectResourceVersion(
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

  public getContactPoint(id: string): Observable<ContactPoint> {
    return this.http.get<ContactPoint>(`${this.baseUrl}/contact-points/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public getContactPoints(): Observable<ContactPoint[]> {
    return this.http.get<ContactPoint[]>(`${this.baseUrl}/contact-points`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public createContactPoint(payload: CreateContactPointRequest): Observable<ContactPoint> {
    return this.http.post<ContactPoint>(`${this.baseUrl}/contact-points`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public replaceContactPoint(id: string, payload: ReplaceContactPointRequest): Observable<ContactPoint> {
    return this.http.put<ContactPoint>(`${this.baseUrl}/contact-points/${encodeURIComponent(id)}`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public deleteContactPoint(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/contact-points/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public getOrganizations(): Observable<ResourceVersionReference[]> {
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

  public getCanonicalOrganizations(): Observable<CanonicalOrganization[]> {
    return this.http.get<CanonicalOrganization[]>(`${this.baseUrl}/organizations`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public createOrganization(payload: CreateOrganizationRequest): Observable<CanonicalOrganization> {
    return this.http.post<CanonicalOrganization>(`${this.baseUrl}/organizations`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public replaceOrganization(id: string, payload: ReplaceOrganizationRequest): Observable<CanonicalOrganization> {
    return this.http.put<CanonicalOrganization>(`${this.baseUrl}/organizations/${encodeURIComponent(id)}`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public deleteOrganization(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/organizations/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public getGroupCategories(groupId: string): Observable<Category[]> {
    return this.http.get<Category[]>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/categories`);
  }

  public createGroupCategory(groupId: string, payload: CreateCategoryRequest): Observable<Category> {
    return this.http.post<Category>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/categories`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public replaceGroupCategory(groupId: string, categoryId: string, payload: ReplaceCategoryRequest): Observable<Category> {
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

  public deleteGroupCategory(groupId: string, categoryId: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/categories/${encodeURIComponent(categoryId)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public getGroups(): Observable<Group[]> {
    return this.http.get<Group[]>(`${this.baseUrl}/groups`);
  }

  public getConcepts(): Observable<Concept[]> {
    return this.http.get<Concept[]>(`${this.baseUrl}/concepts`);
  }

  public createGroup(payload: CreateGroupRequest): Observable<Group> {
    return this.http.post<Group>(`${this.baseUrl}/groups`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public replaceGroup(id: string, payload: ReplaceGroupRequest): Observable<Group> {
    return this.http.put<Group>(`${this.baseUrl}/groups/${encodeURIComponent(id)}`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public deleteGroup(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/groups/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public listPermissions(): Observable<Permission[]> {
    return this.http.get<Permission[]>(`${this.baseUrl}/permissions`);
  }

  public listNotificationTopics(): Observable<NotificationTopic[]> {
    return this.http.get<NotificationTopic[]>(`${this.baseUrl}/notification-topics`);
  }

  public listUserNotificationSubscriptions(userId: string): Observable<NotificationSubscription[]> {
    return this.http.get<NotificationSubscription[]>(
      `${this.baseUrl}/users/${encodeURIComponent(userId)}/notification-subscriptions`,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  public setUserNotificationSubscription(userId: string, topicKey: GlobalNotificationTopicKey, enabled: boolean): Observable<void> {
    const url = `${this.baseUrl}/users/${encodeURIComponent(userId)}/notification-subscriptions/${encodeURIComponent(topicKey)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  public setGroupUserNotificationSubscription(
    groupId: string,
    userId: string,
    topicKey: GroupNotificationTopicKey,
    enabled: boolean
  ): Observable<void> {
    const url = `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/notification-subscriptions/${encodeURIComponent(userId)}/${encodeURIComponent(topicKey)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  public getGroupMemberships(groupId: string): Observable<GroupMembership[]> {
    return this.http.get<GroupMembership[]>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public requestGroupMembership(groupId: string, payload: CreateMembershipRequest): Observable<GroupMembership> {
    return this.http.post<GroupMembership>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/membership-requests`,
      payload,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  public withdrawGroupMembershipRequest(groupId: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/membership-requests`, {
      headers: { Authorization: `Bearer ${this.bearerToken}` }
    });
  }

  public requestGroupRole(groupId: string, payload: CreateRoleRequest): Observable<RoleRequest> {
    return this.http.post<RoleRequest>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/role-requests`,
      payload,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  public leaveOwnGroupMembership(groupId: string): Observable<void> {
    return this.http.delete<void>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/self`,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  public removeGroupMember(groupId: string, userId: string): Observable<void> {
    return this.http.delete<void>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/${encodeURIComponent(userId)}`,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  public decideGroupMembership(groupId: string, userId: string, decision: Decision): Observable<GroupMembership> {
    return this.http.post<GroupMembership>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/${encodeURIComponent(userId)}/${decision}`,
      undefined,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  public decideGroupRoleRequest(groupId: string, requestId: string, decision: Decision): Observable<GroupMembership> {
    return this.http.post<GroupMembership>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/role-requests/${encodeURIComponent(requestId)}/${decision}`,
      undefined,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  public getUsers(): Observable<UserListItem[]> {
    return this.http.get<UserListItem[]>(`${this.baseUrl}/users`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public getUser(id: string): Observable<User> {
    return this.http.get<User>(`${this.baseUrl}/users/${encodeURIComponent(id)}`, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public blockUser(id: string, request: BlockUserRequest): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/users/${encodeURIComponent(id)}/block`, request, {
      headers: { Authorization: `Bearer ${this.bearerToken}` }
    });
  }

  public unblockUser(id: string): Observable<void> {
    return this.http.post<void>(`${this.baseUrl}/users/${encodeURIComponent(id)}/unblock`, undefined, {
      headers: { Authorization: `Bearer ${this.bearerToken}` }
    });
  }
  // Grant or revoke global role
  public setGlobalRole(userId: string, role: GlobalRole, enabled: boolean): Observable<void> {
    const url = `${this.baseUrl}/users/${encodeURIComponent(userId)}/global-roles/${encodeURIComponent(role)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }
  // Grant or revoke global permission
  public setGlobalPermission(userId: string, permission: GlobalPermission, enabled: boolean): Observable<void> {
    const url = `${this.baseUrl}/users/${encodeURIComponent(userId)}/global-permissions/${encodeURIComponent(permission)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  public setGroupMembershipRole(groupId: string, userId: string, role: MembershipRole, enabled: boolean): Observable<void> {
    const url = `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/${encodeURIComponent(userId)}/roles/${encodeURIComponent(role)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  public setGroupMembershipPermission(groupId: string, userId: string, permission: GroupPermission, enabled: boolean): Observable<void> {
    const url = `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/memberships/${encodeURIComponent(userId)}/permissions/${encodeURIComponent(permission)}`;
    const options = { headers: { Authorization: `Bearer ${this.bearerToken}` } };
    return enabled ? this.http.put<void>(url, undefined, options) : this.http.delete<void>(url, options);
  }

  public createResourceDraft(groupId: string, payload: CreateResourceDraftRequest): Observable<WriteResult> {
    return this.http.post<WriteResult>(`${this.baseUrl}/groups/${encodeURIComponent(groupId)}/resources`, payload, {
      headers: {
        Authorization: `Bearer ${this.bearerToken}`
      }
    });
  }

  public copyResource(groupId: string, resourceType: DiscoveryResourceType, resourceId: string): Observable<WriteResult> {
    return this.http.post<WriteResult>(
      `${this.baseUrl}/groups/${encodeURIComponent(groupId)}/resources/${encodeURIComponent(resourceType)}/${encodeURIComponent(resourceId)}/copy`,
      undefined,
      { headers: { Authorization: `Bearer ${this.bearerToken}` } }
    );
  }

  private toHttpParams(params: DiscoverySearchParams): HttpParams {
    let httpParams = new HttpParams();

    if (params.q) {
      httpParams = httpParams.set('q', params.q);
    }

    if (params.editorId) {
      httpParams = httpParams.set('editorId', params.editorId);
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
