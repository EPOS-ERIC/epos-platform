import { Injectable, computed, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';

import type { User } from '../../generated/backofficeSchemas';
import { ApiService } from '../services/api.service';
import { AuthService } from '../services/auth.service';

@Injectable({ providedIn: 'root' })
export class CurrentUserStore {
  private readonly apiService = inject(ApiService);
  private readonly authService = inject(AuthService);
  private initialization?: Promise<void>;

  readonly user = signal<User | null>(null);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);
  readonly isAdmin = computed(() => this.user()?.global_roles.includes('ADMIN') ?? false);

  initialize(): Promise<void> {
    this.initialization ??= this.loadCurrentUser();
    return this.initialization;
  }

  refresh(): Promise<void> {
    return this.loadCurrentUser();
  }

  private async loadCurrentUser(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      await this.authService.initialize();
      if (!this.authService.authenticated()) return;

      this.user.set(await firstValueFrom(this.apiService.getUser('self')));
    } catch (error) {
      console.error('Failed to load current user', error);
      this.error.set('Failed to load current user.');
    } finally {
      this.loading.set(false);
    }
  }
}
