const DEFAULT_REDIRECT = '/dashboard';

/**
 * Validates a candidate redirect path against a strict allowlist shape to
 * prevent open-redirect attacks.
 *
 * A path is only accepted when it:
 * - is a string,
 * - starts with a single `/`,
 * - is not protocol-relative (`//` or `/\`),
 * - and has no URL scheme (e.g. `https:`, `javascript:`).
 *
 * Anything else falls back to `/dashboard`.
 *
 * @param path - The candidate redirect path (untrusted input).
 * @returns The path if it is a safe internal path, otherwise `/dashboard`.
 *
 * @example
 * safeRedirect('/projects/42'); // '/projects/42'
 * safeRedirect('//evil.com');   // '/dashboard'
 */
export function safeRedirect(path: unknown): string {
  if (typeof path !== 'string' || path.length === 0) {
    return DEFAULT_REDIRECT;
  }

  // Must start with a single `/` (rejects relative paths and empty strings).
  if (path[0] !== '/') {
    return DEFAULT_REDIRECT;
  }

  // Reject protocol-relative URLs: `//evil.com` and the backslash variant `/\evil.com`.
  if (path[1] === '/' || path[1] === '\\') {
    return DEFAULT_REDIRECT;
  }

  // Reject anything carrying a URL scheme (e.g. `javascript:`, `https:`).
  if (/^[a-z][a-z0-9+.-]*:/i.test(path)) {
    return DEFAULT_REDIRECT;
  }

  return path;
}
