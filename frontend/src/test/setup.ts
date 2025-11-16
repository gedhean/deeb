import { expect, afterEach, beforeAll, afterAll, vi } from 'vitest';
import { cleanup } from '@testing-library/react';
import * as matchers from '@testing-library/jest-dom/matchers';
import { mutate } from 'swr';

// Extend Vitest's expect with jest-dom matchers
expect.extend(matchers);

// Suppress console.error in tests to avoid cluttering test output
// This is especially useful for intentional error cases in error handling tests
const originalConsoleError = console.error;
beforeAll(() => {
  console.error = vi.fn();
});

afterAll(() => {
  console.error = originalConsoleError;
});

// Cleanup after each test
afterEach(() => {
  cleanup();
  // Clear SWR cache between tests to prevent cross-test contamination
  // Using mutate() without arguments clears all caches
  mutate(() => true, undefined, { revalidate: false });
});
