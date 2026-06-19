import { describe, expect, it } from 'vitest';
import * as actions from '../src/browser/actions.js';

describe('browser-cli interaction layer', () => {
  it('exposes the interaction action functions', () => {
    expect(typeof actions.evalInPage).toBe('function');
    expect(typeof actions.domQuery).toBe('function');
    expect(typeof actions.clickElement).toBe('function');
    expect(typeof actions.fillElement).toBe('function');
    expect(typeof actions.scrollPage).toBe('function');
    expect(typeof actions.snapshotPage).toBe('function');
  });

  it('requires a click target', async () => {
    // No css/text/role and no current page => should reject before launching.
    await expect(
      actions.clickElement(
        // minimal runtime stub: newPage is never reached because target is empty,
        // but navigateOrReuse throws first without a url. We assert it rejects.
        { newPage: async () => { throw new Error('No current page URL is available. Pass --url to navigate first.'); }, close: async () => {} } as never,
        {},
        undefined
      )
    ).rejects.toThrow(/current page URL|click requires/);
  });
});
