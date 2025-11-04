/**
 * Authentication utilities
 *
 * Note: JWT tokens are stored in HttpOnly secure cookies by the backend.
 * These cookies cannot be accessed via JavaScript for security reasons.
 * The backend handles token management through the /api/v1/auth/login and
 * /api/v1/auth/logout endpoints.
 */

/**
 * Check if user is authenticated
 * Since tokens are in HttpOnly cookies, we check by making a request
 * to the backend. This is typically handled by the auth context.
 *
 * @returns Promise that resolves to true if authenticated, false otherwise
 */
export async function checkAuthStatus(): Promise<boolean> {
  try {
    // Try to fetch current user - if this succeeds, user is authenticated
    const response = await fetch(
      `${import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'}/api/v1/auth/me`,
      {
        credentials: 'include', // Include HttpOnly cookies
      },
    );
    return response.ok;
  } catch {
    return false;
  }
}
