import { DestroyRef, Injectable, inject, signal } from '@angular/core';

export type ToastTone = 'success' | 'error' | 'info';

export type ToastState = {
  message: string;
  tone: ToastTone;
  visible: boolean;
};

@Injectable({ providedIn: 'root' })
export class ToastService {
  private readonly destroyRef = inject(DestroyRef);
  private readonly toastState = signal<ToastState | null>(null);
  private timeout?: ReturnType<typeof setTimeout>;

  readonly state = this.toastState.asReadonly();

  constructor() {
    this.destroyRef.onDestroy(() => this.clearTimeout());
  }

  success(message: string): void {
    this.show(message, 'success');
  }

  error(message: string): void {
    this.show(message, 'error');
  }

  info(message: string): void {
    this.show(message, 'info');
  }

  show(message: string, tone: ToastTone = 'info', duration = 3500): void {
    this.clearTimeout();
    this.toastState.set({ message, tone, visible: true });
    this.timeout = setTimeout(() => this.dismiss(), duration);
  }

  dismiss(): void {
    const current = this.toastState();

    if (!current) {
      return;
    }

    this.clearTimeout();
    this.toastState.set({ ...current, visible: false });
    this.timeout = setTimeout(() => {
      this.toastState.set(null);
      this.timeout = undefined;
    }, 180);
  }

  private clearTimeout(): void {
    if (this.timeout) {
      clearTimeout(this.timeout);
      this.timeout = undefined;
    }
  }
}
