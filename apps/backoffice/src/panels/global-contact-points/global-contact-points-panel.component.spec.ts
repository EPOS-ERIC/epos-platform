import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { ContactPoint } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { GlobalContactPointsPanelComponent } from './global-contact-points-panel.component';

describe('GlobalContactPointsPanelComponent', () => {
  const contactPoint: ContactPoint = {
    id: 'contact-1',
    email: 'data@example.org'
  };
  const apiService = {
    getContactPoints: vi.fn(),
    createContactPoint: vi.fn(),
    replaceContactPoint: vi.fn(),
    deleteContactPoint: vi.fn()
  };

  beforeEach(() => {
    vi.clearAllMocks();
    apiService.getContactPoints.mockReturnValue(of([contactPoint]));
    TestBed.configureTestingModule({
      imports: [GlobalContactPointsPanelComponent],
      providers: [{ provide: ApiService, useValue: apiService }]
    });
  });

  it('loads and filters contact points by email', () => {
    const fixture = TestBed.createComponent(GlobalContactPointsPanelComponent);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.textContent).toContain('data@example.org');

    const search = element.querySelector<HTMLInputElement>('input[type="search"]')!;
    search.value = 'unrelated';
    search.dispatchEvent(new Event('input'));
    fixture.detectChanges();

    expect(element.textContent).not.toContain('data@example.org');
    expect(element.textContent).toContain('No contact points match the search.');
  });

  it('paginates contact points in groups of ten and resets the page when searching', async () => {
    const contactPoints = Array.from({ length: 12 }, (_, index): ContactPoint => ({
      id: `contact-${index + 1}`,
      email: `contact-${String(index + 1).padStart(2, '0')}@example.org`
    }));
    apiService.getContactPoints.mockReturnValue(of(contactPoints));
    const fixture = TestBed.createComponent(GlobalContactPointsPanelComponent);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.querySelectorAll('.contact-point-item')).toHaveLength(10);
    expect(element.textContent).not.toContain('contact-11@example.org');

    clickButton(element, 'Next');
    fixture.detectChanges();
    expect(element.querySelectorAll('.contact-point-item')).toHaveLength(2);
    expect(element.textContent).toContain('contact-11@example.org');

    const pageSize = element.querySelector<HTMLSelectElement>('.page-size-control select')!;
    pageSize.value = '25';
    pageSize.dispatchEvent(new Event('change'));
    fixture.detectChanges();
    expect(element.querySelectorAll('.contact-point-item')).toHaveLength(12);
    expect(element.textContent).toContain('Page 1 of 1');

    const search = element.querySelector<HTMLInputElement>('input[type="search"]')!;
    search.value = 'contact-01';
    search.dispatchEvent(new Event('input', { bubbles: true }));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(element.textContent).toContain('Page 1 of 1');
    expect(element.textContent).toContain('contact-01@example.org');
  });

  it('creates a contact point from the add dialog', async () => {
    const created: ContactPoint = { id: 'contact-2', email: 'help@example.org' };
    apiService.createContactPoint.mockReturnValue(of(created));
    const fixture = TestBed.createComponent(GlobalContactPointsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Add New Contact Point');
    fixture.detectChanges();
    await fixture.whenStable();
    setInput(fixture.nativeElement, 'email', created.email);
    await fixture.whenStable();
    fixture.detectChanges();
    submitForm(fixture.nativeElement);
    fixture.detectChanges();

    expect(apiService.createContactPoint).toHaveBeenCalledWith({ email: created.email });
    expect(fixture.nativeElement.textContent).toContain(created.email);
  });

  it('updates an existing contact point from the edit dialog', async () => {
    const updated: ContactPoint = { ...contactPoint, email: 'updated@example.org' };
    apiService.replaceContactPoint.mockReturnValue(of(updated));
    const fixture = TestBed.createComponent(GlobalContactPointsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Edit');
    fixture.detectChanges();
    await fixture.whenStable();
    setInput(fixture.nativeElement, 'email', updated.email);
    await fixture.whenStable();
    fixture.detectChanges();
    submitForm(fixture.nativeElement);
    fixture.detectChanges();

    expect(apiService.replaceContactPoint).toHaveBeenCalledWith(contactPoint.id, {
      email: updated.email
    });
    expect(fixture.nativeElement.textContent).toContain(updated.email);
  });

  it('deletes a contact point after confirmation', () => {
    apiService.deleteContactPoint.mockReturnValue(of(undefined));
    const fixture = TestBed.createComponent(GlobalContactPointsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Remove');
    fixture.detectChanges();
    clickButton(fixture.nativeElement, 'Remove contact point');
    fixture.detectChanges();

    expect(apiService.deleteContactPoint).toHaveBeenCalledWith(contactPoint.id);
    expect(fixture.nativeElement.textContent).not.toContain(contactPoint.email);
  });

  function clickButton(element: HTMLElement, label: string): void {
    const button = Array.from(element.querySelectorAll<HTMLButtonElement>('button'))
      .find((candidate) => candidate.textContent?.replace(/\s+/g, ' ').trim().includes(label));
    expect(button).toBeDefined();
    button!.click();
  }

  function setInput(element: HTMLElement, name: string, value: string): void {
    const input = element.querySelector<HTMLInputElement>(`input[name="${name}"]`)!;
    input.value = value;
    input.dispatchEvent(new Event('input', { bubbles: true }));
  }

  function submitForm(element: HTMLElement): void {
    element.querySelector<HTMLFormElement>('form')!.dispatchEvent(new Event('submit'));
  }
});
