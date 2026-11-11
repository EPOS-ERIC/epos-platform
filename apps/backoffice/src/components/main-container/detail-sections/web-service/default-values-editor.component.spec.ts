import { TestBed } from '@angular/core/testing';

import { DefaultValuesEditorComponent } from './default-values-editor.component';

describe('DefaultValuesEditorComponent', () => {
  it('renders one input per value and allows adding another value', async () => {
    await TestBed.configureTestingModule({ imports: [DefaultValuesEditorComponent] }).compileComponents();
    const fixture = TestBed.createComponent(DefaultValuesEditorComponent);
    const emissions: string[][] = [];
    fixture.componentRef.setInput('values', ['first', 'second']);
    fixture.componentRef.setInput('allowMultiple', true);
    fixture.componentInstance.valuesChange.subscribe((values) => emissions.push(values));
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const inputs = fixture.nativeElement.querySelectorAll('.default-value-row input') as NodeListOf<HTMLInputElement>;
    expect(inputs.length).toBe(2);
    expect(Array.from(inputs).map((input) => input.value)).toEqual(['first', 'second']);

    inputs[1].value = 'updated';
    inputs[1].dispatchEvent(new Event('input'));
    await fixture.whenStable();
    expect(emissions.at(-1)).toEqual(['first', 'updated']);

    (fixture.nativeElement.querySelector('.default-value-add') as HTMLButtonElement).click();
    expect(emissions.at(-1)).toEqual(['first', 'second', '']);
  });

  it('allows only one value when multiple values are disabled', async () => {
    await TestBed.configureTestingModule({ imports: [DefaultValuesEditorComponent] }).compileComponents();
    const fixture = TestBed.createComponent(DefaultValuesEditorComponent);
    fixture.componentRef.setInput('values', ['first', 'second']);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('.default-value-row input').length).toBe(1);
    expect(fixture.nativeElement.querySelector('.default-value-add')).toBeNull();
  });
});
