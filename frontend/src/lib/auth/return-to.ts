const RETURN_TO_KEY = 'deeb.returnTo';

/**
 * Stores the path the user should be returned to after authentication.
 *
 * Backed by `sessionStorage` so it survives the same-tab cross-origin OAuth
 * roundtrip (SPA → backend → Google → callback → SPA).
 *
 * @param path - The path to return to after login.
 */
export function setReturnTo(path: string): void {
  sessionStorage.setItem(RETURN_TO_KEY, path);
}

/**
 * Reads the stored return-to path.
 *
 * @returns The stored path, or `null` when nothing is stored.
 */
export function getReturnTo(): string | null {
  return sessionStorage.getItem(RETURN_TO_KEY);
}

/**
 * Removes the stored return-to path.
 */
export function clearReturnTo(): void {
  sessionStorage.removeItem(RETURN_TO_KEY);
}
