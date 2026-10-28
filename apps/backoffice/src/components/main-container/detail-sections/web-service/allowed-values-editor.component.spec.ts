import { TestBed } from '@angular/core/testing';

import type { DatasetEnumOptionForm } from '../../../../models/dataset-detail-form';
import { ToastService } from '../../../../services/toast.service';
import { AllowedValuesEditorComponent } from './allowed-values-editor.component';

describe('AllowedValuesEditorComponent', () => {
  it('filters values and adds comma or newline separated options', async () => {
    await TestBed.configureTestingModule({
      imports: [AllowedValuesEditorComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(AllowedValuesEditorComponent);
    const emittedValues: DatasetEnumOptionForm[][] = [];

    fixture.componentRef.setInput('values', [
      { value: 'First option', label: 'First label' },
      { value: 'Second option', label: 'Second label' },
    ]);
    fixture.componentInstance.valuesChange.subscribe((values) => emittedValues.push(values));
    fixture.detectChanges();

    const searchInput = fixture.nativeElement.querySelector('.allowed-values-search input') as HTMLInputElement;
    searchInput.value = 'second';
    searchInput.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('.allowed-value-row').length).toBe(1);
    expect(fixture.nativeElement.querySelector('.allowed-value-input')?.value).toBe('Second option');

    (fixture.nativeElement.querySelector('.allowed-values-add-toggle') as HTMLButtonElement).click();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.allowed-values-add-menu button') as HTMLButtonElement).click();
    fixture.detectChanges();

    const textarea = fixture.nativeElement.querySelector('.allowed-values-textarea') as HTMLTextAreaElement;
    textarea.value = 'Third option, Fourth option\nFifth option';
    textarea.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.resource-dialog-actions .app-button--primary') as HTMLButtonElement).click();

    expect(emittedValues.at(-1)).toEqual([
      { value: 'First option', label: 'First label' },
      { value: 'Second option', label: 'Second label' },
      { value: 'Third option', label: '' },
      { value: 'Fourth option', label: '' },
      { value: 'Fifth option', label: '' },
    ]);
  });

  it('rejects duplicate values from single edits and bulk additions', async () => {
    const toast = { error: vi.fn() };
    await TestBed.configureTestingModule({
      imports: [AllowedValuesEditorComponent],
      providers: [{ provide: ToastService, useValue: toast }]
    }).compileComponents();
    const fixture = TestBed.createComponent(AllowedValuesEditorComponent);
    const emittedValues: DatasetEnumOptionForm[][] = [];

    fixture.componentRef.setInput('values', [
      { value: 'First option', label: '' },
      { value: 'Second option', label: '' },
    ]);
    fixture.componentInstance.valuesChange.subscribe((values) => emittedValues.push(values));
    fixture.detectChanges();

    const secondOption = fixture.nativeElement.querySelectorAll('.allowed-value-input')[1] as HTMLInputElement;
    secondOption.value = 'First option';
    secondOption.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    expect(emittedValues).toHaveLength(0);
    expect(secondOption.value).toBe('Second option');
    expect(toast.error).toHaveBeenCalledWith('The option "First option" already exists.');

    (fixture.nativeElement.querySelector('.allowed-values-add-toggle') as HTMLButtonElement).click();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.allowed-values-add-menu button') as HTMLButtonElement).click();
    fixture.detectChanges();
    const textarea = fixture.nativeElement.querySelector('.allowed-values-textarea') as HTMLTextAreaElement;
    textarea.value = 'Third option, Third option';
    textarea.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.resource-dialog-actions .app-button--primary') as HTMLButtonElement).click();
    fixture.detectChanges();

    expect(emittedValues).toHaveLength(0);
    expect(fixture.nativeElement.querySelector('.allowed-values-error')?.textContent).toContain('Third option');
    expect(toast.error).toHaveBeenCalledWith('The option "Third option" already exists.');
  });
});
