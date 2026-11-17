import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import type { SoftwareApplicationResourceMetadata } from '../../../../../generated/backofficeSchemas';
import { ApiService } from '../../../../services/api.service';
import { SoftwareGeneralInformationComponent } from './software-general-information.component';

describe('SoftwareGeneralInformationComponent', () => {
  const apiService = {
    getOrganizations: vi.fn(() => of([
      { id: 'creator-id', name: 'Creator Organization' }
    ]))
  };

  beforeEach(() => {
    vi.clearAllMocks();
    TestBed.configureTestingModule({
      imports: [SoftwareGeneralInformationComponent],
      providers: [{ provide: ApiService, useValue: apiService }]
    });
  });

  it('renders creator names without exposing their identifiers', () => {
    const fixture = TestBed.createComponent(SoftwareGeneralInformationComponent);
    fixture.componentRef.setInput('model', applicationMetadata());
    fixture.componentRef.setInput('disabled', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Creator Organization');
    expect(fixture.nativeElement.textContent).not.toContain('creator-id');
    expect(apiService.getOrganizations).not.toHaveBeenCalled();
  });

  it('selects creators as resource references from the organizations catalog', () => {
    const fixture = TestBed.createComponent(SoftwareGeneralInformationComponent);
    const metadata = applicationMetadata([]);
    fixture.componentRef.setInput('model', metadata);
    fixture.detectChanges();

    const element = fixture.nativeElement as HTMLElement;
    const addButton = Array.from(element.querySelectorAll<HTMLButtonElement>('button'))
      .find((button) => button.textContent?.trim() === 'Add Creator');
    addButton?.click();

    expect(metadata.creators).toEqual([{ id: 'creator-id', name: 'Creator Organization' }]);
  });
});

function applicationMetadata(creators = [{ id: 'creator-id', name: 'Creator Organization' }]): SoftwareApplicationResourceMetadata {
  return {
    resource_type: 'software-application',
    name: 'Application',
    description: 'Description',
    keywords: [],
    creators,
    identifiers: [],
    contact_points: [],
    categories: []
  };
}
