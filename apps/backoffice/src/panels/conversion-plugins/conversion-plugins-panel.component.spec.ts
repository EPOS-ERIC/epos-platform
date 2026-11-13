import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { vi } from 'vitest';

import type { ConversionRelation, ManagedConversionPlugin } from '../../../generated/backofficeSchemas';
import { ApiService } from '../../services/api.service';
import { ConversionPluginsPanelComponent } from './conversion-plugins-panel.component';

describe('ConversionPluginsPanelComponent', () => {
  const plugins: ManagedConversionPlugin[] = [
    {
      id: 'plugin-2',
      name: 'Tabular converter',
      description: 'Converts tabular data.',
      repository: 'https://example.test/tabular-converter',
      branch: 'main',
      runtime: 'python',
      arguments: [],
      enabled: true,
      installed: true,
      installed_commit: 'abcdef1234567890',
      created_by: 'user-1',
      updated_by: 'user-1'
    },
    {
      id: 'plugin-1',
      name: 'Archive extractor',
      description: '',
      repository: 'https://example.test/archive-extractor',
      branch: 'stable',
      runtime: 'binary',
      arguments: ['--safe'],
      enabled: false,
      installed: false,
      installed_commit: null,
      created_by: 'user-1',
      updated_by: 'user-2'
    }
  ];
  const relations: ConversionRelation[] = [
    {
      id: 'relation-1',
      distribution_id: 'distribution-a',
      plugin_id: 'plugin-1',
      input_format: 'application/zip',
      output_format: 'text/csv',
      created_by: 'user-1',
      updated_by: 'user-1'
    },
    {
      id: 'relation-2',
      distribution_id: 'distribution-a',
      plugin_id: 'plugin-1',
      input_format: 'application/zip',
      output_format: 'application/json',
      created_by: 'user-1',
      updated_by: 'user-1'
    },
    {
      id: 'relation-3',
      distribution_id: 'distribution-b',
      plugin_id: 'plugin-1',
      input_format: 'application/gzip',
      output_format: 'text/csv',
      created_by: 'user-1',
      updated_by: 'user-1'
    }
  ];
  const apiService = {
    listConversionPlugins: vi.fn(),
    listConversionRelations: vi.fn(),
    createConversionPlugin: vi.fn(),
    replaceConversionPlugin: vi.fn(),
    syncConversionPlugin: vi.fn(),
    deleteConversionPlugin: vi.fn()
  };

  beforeEach(() => {
    vi.clearAllMocks();
    apiService.listConversionPlugins.mockReturnValue(of(plugins));
    apiService.listConversionRelations.mockReturnValue(of(relations));
    apiService.createConversionPlugin.mockReturnValue(of(plugins[0]));
    apiService.replaceConversionPlugin.mockReturnValue(of(plugins[0]));
    apiService.syncConversionPlugin.mockReturnValue(of(plugins[0]));
    apiService.deleteConversionPlugin.mockReturnValue(of(undefined));
    TestBed.configureTestingModule({
      imports: [ConversionPluginsPanelComponent],
      providers: [{ provide: ApiService, useValue: apiService }]
    });
  });

  it('loads and renders conversion plugins', () => {
    const fixture = TestBed.createComponent(ConversionPluginsPanelComponent);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(apiService.listConversionPlugins).toHaveBeenCalledOnce();
    expect(apiService.listConversionRelations).toHaveBeenCalledOnce();
    expect(element.querySelectorAll('.plugin-item')).toHaveLength(2);
    expect(element.textContent).toContain('Archive extractor');
    expect(element.textContent).toContain('Not installed');
    expect(element.textContent).toContain('Commit: abcdef12');
    expect(element.textContent).toContain('Distributions (2)');
  });

  it('shows relations grouped by distribution', () => {
    const fixture = TestBed.createComponent(ConversionPluginsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Distributions (2)');
    fixture.detectChanges();

    const dialog = fixture.nativeElement.querySelector('.relations-dialog') as HTMLElement;
    expect(dialog.textContent).toContain('2 distributions');
    expect(dialog.textContent).toContain('3 relations');
    expect(dialog.textContent).toContain('distribution-a');
    expect(dialog.textContent).toContain('application/zip');
    expect(dialog.textContent).toContain('application/json');
  });

  it('filters plugins by runtime', async () => {
    const fixture = TestBed.createComponent(ConversionPluginsPanelComponent);
    fixture.detectChanges();
    const input = fixture.nativeElement.querySelector('input[type="search"]') as HTMLInputElement;

    input.value = 'python';
    input.dispatchEvent(new Event('input', { bubbles: true }));
    await fixture.whenStable();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelectorAll('.plugin-item')).toHaveLength(1);
    expect(fixture.nativeElement.textContent).toContain('Tabular converter');
  });

  it('shows an error when loading fails', () => {
    apiService.listConversionPlugins.mockReturnValue(throwError(() => new Error('Unavailable')));
    const fixture = TestBed.createComponent(ConversionPluginsPanelComponent);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Failed to load conversion plugins');
  });

  it('registers a conversion plugin', () => {
    const created = { ...plugins[0], id: 'plugin-3', name: 'New converter' };
    apiService.createConversionPlugin.mockReturnValue(of(created));
    const fixture = TestBed.createComponent(ConversionPluginsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Add New Plugin');
    fixture.detectChanges();
    const component = editableComponent(fixture.componentInstance);
    component.pluginForm.name = created.name;
    component.pluginForm.repository = created.repository;
    component.savePlugin();
    fixture.detectChanges();

    expect(apiService.createConversionPlugin).toHaveBeenCalledWith({
      name: created.name,
      description: '',
      repository: created.repository,
      branch: 'main',
      runtime: 'python',
      arguments: []
    });
    expect(fixture.nativeElement.textContent).toContain(created.name);
  });

  it('replaces an existing conversion plugin', () => {
    const updated = { ...plugins[1], name: 'Updated extractor' };
    apiService.replaceConversionPlugin.mockReturnValue(of(updated));
    const fixture = TestBed.createComponent(ConversionPluginsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Edit');
    fixture.detectChanges();
    const component = editableComponent(fixture.componentInstance);
    component.pluginForm.name = updated.name;
    component.savePlugin();
    fixture.detectChanges();

    expect(apiService.replaceConversionPlugin).toHaveBeenCalledWith(plugins[1].id, {
      name: updated.name,
      description: plugins[1].description,
      repository: plugins[1].repository,
      branch: plugins[1].branch,
      runtime: plugins[1].runtime,
      arguments: plugins[1].arguments,
      enabled: plugins[1].enabled
    });
  });

  it('synchronizes a conversion plugin', () => {
    const synced = { ...plugins[1], installed: true, installed_commit: '1234567890abcdef' };
    apiService.syncConversionPlugin.mockReturnValue(of(synced));
    const fixture = TestBed.createComponent(ConversionPluginsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Sync');
    fixture.detectChanges();

    expect(apiService.syncConversionPlugin).toHaveBeenCalledWith(plugins[1].id);
    expect(fixture.nativeElement.textContent).toContain('Commit: 12345678');
  });

  it('removes a conversion plugin after confirmation', () => {
    const fixture = TestBed.createComponent(ConversionPluginsPanelComponent);
    fixture.detectChanges();

    clickButton(fixture.nativeElement, 'Remove');
    fixture.detectChanges();
    clickButton(fixture.nativeElement, 'Remove plugin');
    fixture.detectChanges();

    expect(apiService.deleteConversionPlugin).toHaveBeenCalledWith(plugins[1].id);
    expect(fixture.nativeElement.textContent).not.toContain(plugins[1].name);
  });

  function clickButton(element: HTMLElement, label: string): void {
    const button = Array.from(element.querySelectorAll<HTMLButtonElement>('button'))
      .find((candidate) => candidate.textContent?.replace(/\s+/g, ' ').trim().includes(label));
    expect(button).toBeDefined();
    button!.click();
  }

  function editableComponent(component: ConversionPluginsPanelComponent): {
    pluginForm: { name: string; repository: string };
    savePlugin(): void;
  } {
    return component as unknown as {
      pluginForm: { name: string; repository: string };
      savePlugin(): void;
    };
  }
});
