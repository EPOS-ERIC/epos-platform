import { Injectable, signal } from '@angular/core';

type DialogBase = {
  eyebrow?: string;
  title: string;
  message?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  destructive?: boolean;
};

export type ConfirmDialog = DialogBase & {
  kind: 'confirm';
};

export type PromptDialog = DialogBase & {
  kind: 'prompt';
  inputLabel: string;
  placeholder?: string;
  initialValue?: string;
  required?: boolean;
};

export type DialogState = ConfirmDialog | PromptDialog;

@Injectable({ providedIn: 'root' })
export class DialogService {
  private readonly dialogState = signal<DialogState | null>(null);
  private resolve?: (value: boolean | string | null) => void;

  readonly state = this.dialogState.asReadonly();

  confirm(config: Omit<ConfirmDialog, 'kind'>): Promise<boolean> {
    return this.open<boolean>({ ...config, kind: 'confirm' });
  }

  prompt(config: Omit<PromptDialog, 'kind'>): Promise<string | null> {
    return this.open<string | null>({ ...config, kind: 'prompt' });
  }

  accept(value: string = ''): void {
    const dialog = this.dialogState();

    if (!dialog) {
      return;
    }

    this.finish(dialog.kind === 'confirm' ? true : value);
  }

  dismiss(): void {
    const dialog = this.dialogState();

    if (dialog) {
      this.finish(dialog.kind === 'confirm' ? false : null);
    }
  }

  private open<T extends boolean | string | null>(dialog: DialogState): Promise<T> {
    this.dismiss();
    this.dialogState.set(dialog);

    return new Promise<T>((resolve) => {
      this.resolve = resolve as (value: boolean | string | null) => void;
    });
  }

  private finish(value: boolean | string | null): void {
    const resolve = this.resolve;
    this.resolve = undefined;
    this.dialogState.set(null);
    resolve?.(value);
  }
}
