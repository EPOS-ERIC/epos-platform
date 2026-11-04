import { resourceCapabilities } from './resource-capabilities';

describe('resourceCapabilities', () => {
  it('allows editing, saving, submitting and deleting drafts', () => {
    const capabilities = resourceCapabilities('DRAFT', [
      'group.resource.edit',
      'group.resource.submit',
      'group.resource.discard'
    ]);

    expect(capabilities.edit).toBe(true);
    expect(capabilities.save).toBe(true);
    expect(capabilities.submit).toBe(true);
    expect(capabilities.delete).toBe(true);
    expect(capabilities.viewHistory).toBe(true);
    expect(capabilities.publish).toBe(false);
  });

  it('derives review actions independently from their permissions', () => {
    expect(resourceCapabilities('SUBMITTED', []).publish).toBe(false);
    expect(resourceCapabilities('SUBMITTED', []).reject).toBe(false);
    expect(resourceCapabilities('SUBMITTED', []).withdraw).toBe(false);
    expect(resourceCapabilities('SUBMITTED', []).viewHistory).toBe(true);
    expect(resourceCapabilities('SUBMITTED', ['group.resource.publish']).publish).toBe(true);
    expect(resourceCapabilities('SUBMITTED', ['group.resource.publish']).reject).toBe(false);
    expect(resourceCapabilities('SUBMITTED', ['group.resource.reject']).publish).toBe(false);
    expect(resourceCapabilities('SUBMITTED', ['group.resource.reject']).reject).toBe(true);
    expect(resourceCapabilities('SUBMITTED', ['group.resource.withdraw']).withdraw).toBe(true);
  });

  it('maps the remaining lifecycle states to their available transitions', () => {
    expect(resourceCapabilities('PUBLISHED', ['group.resource.unpublish']).archive).toBe(true);
    expect(resourceCapabilities('PUBLISHED', ['group.resource.create']).createDraft).toBe(true);
    expect(resourceCapabilities('ARCHIVED', ['group.resource.create']).createDraft).toBe(true);
    expect(resourceCapabilities('REJECTED', ['group.resource.reopen']).reopen).toBe(true);
    expect(Object.values(resourceCapabilities(null, ['group.resource.publish'])).some(Boolean)).toBe(false);
  });

  it('allows copy only for a published resource with at least one writable target group', () => {
    expect(resourceCapabilities('PUBLISHED', [], true).copy).toBe(true);
    expect(resourceCapabilities('PUBLISHED', [], false).copy).toBe(false);
    expect(resourceCapabilities('DRAFT', [], true).copy).toBe(false);
  });
});
