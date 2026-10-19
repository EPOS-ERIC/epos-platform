import { DestroyRef, Injectable, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { catchError, finalize, from, of, switchMap } from 'rxjs';

import type { MembershipRole } from '../../generated/backofficeSchemas';
import { ApiService } from './api.service';
import { CurrentUserStore } from '../stores/current-user.store';
import { GroupsStore } from '../stores/groups.store';
import { ToastService } from './toast.service';

type RequestableMembershipRole = Extract<MembershipRole, 'EDITOR' | 'REVIEWER'>;

@Injectable({ providedIn: 'root' })
export class MembershipRequestDialogService {
  private readonly apiService = inject(ApiService);
  private readonly currentUserStore = inject(CurrentUserStore);
  private readonly destroyRef = inject(DestroyRef);
  private readonly groupsStore = inject(GroupsStore);
  private readonly toast = inject(ToastService);

  readonly openState = signal(false);
  readonly groupId = signal('');
  readonly role = signal<RequestableMembershipRole>('EDITOR');
  readonly saving = signal(false);
  readonly error = signal('');
  readonly groups = computed(() => {
    const unavailableGroupIds = new Set(
      this.currentUserStore.user()?.memberships
        .filter((membership) => membership.status === 'ACTIVE' || membership.status === 'PENDING')
        .map((membership) => membership.group.id) ?? []
    );
    return this.groupsStore.groups().filter((group) => !unavailableGroupIds.has(group.id));
  });
  readonly groupsLoading = this.groupsStore.loading;
  readonly roles: readonly RequestableMembershipRole[] = ['EDITOR', 'REVIEWER'];

  open(groupId?: string): void {
    if(groupId){
      this.groupId.set(groupId);  
    }
    else{
      this.groupId.set(this.groups()[0]?.id ?? '');
    }
    this.role.set('EDITOR');
    this.error.set('');
    this.openState.set(true);
  }

  close(): void {
    if (this.saving()) return;

    this.openState.set(false);
    this.error.set('');
  }

  submit(): void {
    const groupId = this.groupId();
    const role = this.role();
    if (this.saving() || !groupId || !role) return;

    this.saving.set(true);
    this.error.set('');
    this.apiService
      .requestGroupMembership(groupId, { roles: [role] })
      .pipe(
        switchMap(() => from(this.currentUserStore.refresh())),
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.saving.set(false)),
        catchError((error) => {
          console.error('Failed to request group membership', error);
          this.error.set('Failed to submit the membership request.');
          return of(null);
        })
      )
      .subscribe((result) => {
        if (result === null) return;

        this.openState.set(false);
        this.toast.success('Membership request submitted.');
      });
  }
}
