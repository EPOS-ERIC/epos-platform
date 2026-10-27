import { TestBed } from '@angular/core/testing';
import { vi } from 'vitest';

import { KeywordChipsComponent } from './keyword-chips.component';

describe('KeywordChipsComponent', () => {
  beforeEach(() => {
    TestBed.configureTestingModule({ imports: [KeywordChipsComponent] });
  });

  it('adds trimmed, unique keywords with Enter', () => {
    const fixture = TestBed.createComponent(KeywordChipsComponent);
    fixture.componentRef.setInput('keywords', ['Earth']);
    const emitted = vi.fn();
    fixture.componentInstance.keywordsChange.subscribe(emitted);
    fixture.detectChanges();

    const input = fixture.nativeElement.querySelector('.keyword-input') as HTMLInputElement;
    input.value = ' earth, Science ';
    input.dispatchEvent(new Event('input'));
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));

    expect(emitted).toHaveBeenCalledWith(['Earth', 'Science']);
  });

  it('removes a keyword by index', () => {
    const fixture = TestBed.createComponent(KeywordChipsComponent);
    fixture.componentRef.setInput('keywords', ['Earth', 'Science']);
    const emitted = vi.fn();
    fixture.componentInstance.keywordsChange.subscribe(emitted);
    fixture.detectChanges();

    (fixture.nativeElement.querySelector('.keyword-chip button') as HTMLButtonElement).click();

    expect(emitted).toHaveBeenCalledWith(['Science']);
  });

  it('renders read-only chips without editing controls when disabled', () => {
    const fixture = TestBed.createComponent(KeywordChipsComponent);
    fixture.componentRef.setInput('keywords', ['Earth']);
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Earth');
    expect(fixture.nativeElement.querySelector('input')).toBeNull();
    expect(fixture.nativeElement.querySelector('button')).toBeNull();
  });
});
