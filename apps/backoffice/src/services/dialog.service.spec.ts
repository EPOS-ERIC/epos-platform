import { TestBed } from '@angular/core/testing';

import { DialogService } from './dialog.service';

describe('DialogService', () => {
  let service: DialogService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(DialogService);
  });

  it('resolves confirmations from the outlet response', async () => {
    const result = service.confirm({ title: 'Confirm action' });

    expect(service.state()?.kind).toBe('confirm');
    service.accept();

    await expect(result).resolves.toBe(true);
    expect(service.state()).toBeNull();
  });

  it('resolves dismissed prompts with null', async () => {
    const result = service.prompt({ title: 'Add note', inputLabel: 'Note' });

    service.dismiss();

    await expect(result).resolves.toBeNull();
  });
});
