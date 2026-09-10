import { resourceCapabilities } from './resource-capabilities';

describe('resourceCapabilities', () => {
  it('allows editing, saving, submitting and deleting drafts', () => {
    const capabilities = resourceCapabilities('DRAFT', false);

    expect(capabilities.edit).toBe(true);
    expect(capabilities.save).toBe(true);
    expect(capabilities.submit).toBe(true);
    expect(capabilities.delete).toBe(true);
    expect(capabilities.publish).toBe(false);
  });

  it('allows only admins to review submitted resources', () => {
    expect(resourceCapabilities('SUBMITTED', false).publish).toBe(false);
    expect(resourceCapabilities('SUBMITTED', false).reject).toBe(false);
    expect(resourceCapabilities('SUBMITTED', true).publish).toBe(true);
    expect(resourceCapabilities('SUBMITTED', true).reject).toBe(true);
  });

  it('maps the remaining lifecycle states to their available transitions', () => {
    expect(resourceCapabilities('PUBLISHED', false).archive).toBe(true);
    expect(resourceCapabilities('PUBLISHED', false).createDraft).toBe(true);
    expect(resourceCapabilities('ARCHIVED', false).createDraft).toBe(true);
    expect(resourceCapabilities('REJECTED', false).reopen).toBe(true);
    expect(Object.values(resourceCapabilities(null, true)).some(Boolean)).toBe(false);
  });
});
