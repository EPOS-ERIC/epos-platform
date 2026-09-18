import { resourceCapabilities } from './resource-capabilities';

describe('resourceCapabilities', () => {
  it('allows editing, saving, submitting and deleting drafts', () => {
    const capabilities = resourceCapabilities('DRAFT', [
      'resource.edit',
      'resource.submit',
      'resource.discard'
    ]);

    expect(capabilities.edit).toBe(true);
    expect(capabilities.save).toBe(true);
    expect(capabilities.submit).toBe(true);
    expect(capabilities.delete).toBe(true);
    expect(capabilities.publish).toBe(false);
  });

  it('derives review actions independently from their permissions', () => {
    expect(resourceCapabilities('SUBMITTED', []).publish).toBe(false);
    expect(resourceCapabilities('SUBMITTED', []).reject).toBe(false);
    expect(resourceCapabilities('SUBMITTED', ['resource.publish']).publish).toBe(true);
    expect(resourceCapabilities('SUBMITTED', ['resource.publish']).reject).toBe(false);
    expect(resourceCapabilities('SUBMITTED', ['resource.reject']).publish).toBe(false);
    expect(resourceCapabilities('SUBMITTED', ['resource.reject']).reject).toBe(true);
  });

  it('maps the remaining lifecycle states to their available transitions', () => {
    expect(resourceCapabilities('PUBLISHED', ['resource.unpublish']).archive).toBe(true);
    expect(resourceCapabilities('PUBLISHED', ['resource.create']).createDraft).toBe(true);
    expect(resourceCapabilities('ARCHIVED', ['resource.create']).createDraft).toBe(true);
    expect(resourceCapabilities('REJECTED', ['resource.reopen']).reopen).toBe(true);
    expect(Object.values(resourceCapabilities(null, ['resource.publish'])).some(Boolean)).toBe(false);
  });
});
