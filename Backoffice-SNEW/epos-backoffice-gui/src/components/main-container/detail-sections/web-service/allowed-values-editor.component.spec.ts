import { TestBed } from '@angular/core/testing';

import { AllowedValuesEditorComponent } from './allowed-values-editor.component';

describe('AllowedValuesEditorComponent', () => {
  it('filters values and adds comma or newline separated options', async () => {
    await TestBed.configureTestingModule({
      imports: [AllowedValuesEditorComponent]
    }).compileComponents();
    const fixture = TestBed.createComponent(AllowedValuesEditorComponent);
    const emittedValues: string[][] = [];

    fixture.componentRef.setInput('values', ['First option', 'Second option']);
    fixture.componentInstance.valuesChange.subscribe((values) => emittedValues.push(values));
    fixture.detectChanges();

    const searchInput = fixture.nativeElement.querySelector('.allowed-values-search input') as HTMLInputElement;
    searchInput.value = 'second';
    searchInput.dispatchEvent(new Event('input'));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('.allowed-value-row').length).toBe(1);
    expect(fixture.nativeElement.querySelector('.allowed-value-row input')?.value).toBe('Second option');

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
      'First option',
      'Second option',
      'Third option',
      'Fourth option',
      'Fifth option'
    ]);
  });
});
