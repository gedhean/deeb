import { describe, it, expect, beforeEach } from 'vitest';
import { setReturnTo, getReturnTo, clearReturnTo } from './return-to';

describe('returnTo helpers', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('roundtrips a set value through get', () => {
    setReturnTo('/projects/42');
    expect(getReturnTo()).toBe('/projects/42');
  });

  it('returns null when nothing is stored', () => {
    expect(getReturnTo()).toBeNull();
  });

  it('clear removes the stored value', () => {
    setReturnTo('/projects/42');
    clearReturnTo();
    expect(getReturnTo()).toBeNull();
  });

  it('stores under the deeb.returnTo key', () => {
    setReturnTo('/settings');
    expect(sessionStorage.getItem('deeb.returnTo')).toBe('/settings');
  });

  it('overwrites a previously stored value', () => {
    setReturnTo('/first');
    setReturnTo('/second');
    expect(getReturnTo()).toBe('/second');
  });
});
