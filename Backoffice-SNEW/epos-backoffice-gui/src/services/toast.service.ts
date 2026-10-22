import { DestroyRef, Injectable, inject, signal } from '@angular/core';

export type ToastTone = 'success' | 'error' | 'info';

export type ToastIssue = {
  message: string;
  label?: string;
  reference?: string;
  target?: string;
};

export type ToastOptions = {
  duration?: number | null;
  issues?: readonly ToastIssue[];
};

export type ToastState = {
  message: string;
  tone: ToastTone;
  visible: boolean;
  issues?: readonly ToastIssue[];
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

  error(message: string, options?: ToastOptions): void {
    this.show(message, 'error', options);
  }

  info(message: string): void {
    this.show(message, 'info');
  }

  show(message: string, tone: ToastTone = 'info', options: ToastOptions = {}): void {
    this.clearTimeout();
    const issues = options.issues?.filter((issue) => issue.message.trim());
    this.toastState.set({
      message,
      tone,
      visible: true,
      ...(issues?.length ? { issues } : {})
    });

    const duration = options.duration === undefined ? 3500 : options.duration;
    if (duration !== null) {
      this.timeout = setTimeout(() => this.dismiss(), duration);
    }
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
