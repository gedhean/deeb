import { describe, it, expect } from 'vitest';
import { safeRedirect } from './safe-redirect';

describe('safeRedirect', () => {
  describe('safe internal paths', () => {
    it('accepts a simple path', () => {
      expect(safeRedirect('/projects/42')).toBe('/projects/42');
    });

    it('accepts a path with a query string', () => {
      expect(safeRedirect('/dashboard?tab=x')).toBe('/dashboard?tab=x');
    });

    it('accepts the root path', () => {
      expect(safeRedirect('/')).toBe('/');
    });
  });

  describe('malicious or invalid input falls back to /dashboard', () => {
    it('rejects protocol-relative URLs (//)', () => {
      expect(safeRedirect('//evil.com')).toBe('/dashboard');
    });

    it('rejects protocol-relative URLs with a backslash (/\\)', () => {
      expect(safeRedirect('/\\evil.com')).toBe('/dashboard');
    });

    it('rejects absolute https URLs', () => {
      expect(safeRedirect('https://evil.com')).toBe('/dashboard');
    });

    it('rejects javascript: scheme', () => {
      expect(safeRedirect('javascript:alert(1)')).toBe('/dashboard');
    });

    it('rejects the empty string', () => {
      expect(safeRedirect('')).toBe('/dashboard');
    });

    it('rejects relative paths without a leading slash', () => {
      expect(safeRedirect('projects/42')).toBe('/dashboard');
    });

    it('rejects undefined', () => {
      expect(safeRedirect(undefined)).toBe('/dashboard');
    });

    it('rejects null', () => {
      expect(safeRedirect(null)).toBe('/dashboard');
    });

    it('rejects non-string values', () => {
      expect(safeRedirect(42)).toBe('/dashboard');
      expect(safeRedirect({})).toBe('/dashboard');
    });
  });
});
