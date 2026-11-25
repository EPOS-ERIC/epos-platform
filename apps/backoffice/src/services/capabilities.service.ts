import { HttpClient } from '@angular/common/http';
import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import type { Capabilities } from '../../generated/backofficeSchemas';

@Injectable({ providedIn: 'root' })
export class CapabilitiesService {
  private readonly http = inject(HttpClient);
  private initialization?: Promise<Capabilities>;

  readonly capabilities = signal<Capabilities | null>(null);
  readonly emailNotificationsEnabled = computed(() => this.capabilities()?.notifications.email_enabled ?? false);
  readonly organizationManagement = computed(() => this.capabilities()?.metadata_sync.management ?? null);
  readonly issueTracker = computed(() => this.capabilities()?.issue_tracker ?? null);

  initialize(): Promise<Capabilities> {
    this.initialization ??= firstValueFrom(this.http.get<Capabilities>('api/v1/capabilities'))
      .then((capabilities) => {
        this.capabilities.set(capabilities);
        return capabilities;
      });
    return this.initialization;
  }
}
