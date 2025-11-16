import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Compares two objects deeply and returns a new object containing only the
 * properties that differ between them, with values from the second object.
 *
 * @param obj1 - The first object (original/source)
 * @param obj2 - The second object (updated/target)
 * @returns An object containing only the properties that differ, with values from obj2
 *
 * @example
 * ```ts
 * const o1 = { a: 1, b: 2, c: 3 };
 * const o2 = { a: 1, b: 3, c: 3 };
 * objDiff(o1, o2); // Returns { b: 3 }
 * ```
 *
 * @example
 * ```ts
 * const o1 = { name: 'John', age: 30, settings: { theme: 'dark' } };
 * const o2 = { name: 'Jane', age: 30, settings: { theme: 'dark' } };
 * objDiff(o1, o2); // Returns { name: 'Jane' }
 * ```
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function objDiff<T = Record<string, unknown>>(obj1: T, obj2: any): Partial<T> {
  const diff: Partial<T> = {};

  // Get all keys from both objects
  const allKeys = new Set([
    ...Object.keys(obj1 as Record<string, unknown>),
    ...Object.keys(obj2 as Record<string, unknown>),
  ]);

  for (const key of allKeys) {
    const val1 = (obj1 as Record<string, unknown>)[key];
    const val2 = (obj2 as Record<string, unknown>)[key];

    // Check if values are different
    if (!isEqual(val1, val2)) {
      diff[key as keyof T] = val2 as T[keyof T];
    }
  }

  return diff;
}

/**
 * Compares two objects deeply and returns a new object containing only the
 * properties that differ, excluding undefined values from the result.
 * This is useful when you want to get only the actual changes without including removed properties.
 *
 * @param obj1 - The first object (original/source)
 * @param obj2 - The second object (updated/target)
 * @returns An object containing only the properties that differ (excluding undefined values)
 *
 * @example
 * ```ts
 * const o1 = { a: 1, b: 2, c: 3 };
 * const o2 = { a: 1, b: 4 };
 * objDiffExcludeUndefined(o1, o2); // Returns { b: 4 } (c is excluded because it's undefined in o2)
 * ```
 */
export function objDiffExcludeUndefined<T = Record<string, unknown>>(
  obj1: T,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  obj2: any,
): Partial<T> {
  const diff = objDiff(obj1, obj2);

  // Remove undefined values
  return Object.fromEntries(
    Object.entries(diff).filter(([_, value]) => value !== undefined),
  ) as Partial<T>;
}

/**
 * Deep equality check for two values
 */
function isEqual(val1: unknown, val2: unknown): boolean {
  // Handle same reference or primitive equality
  if (val1 === val2) return true;

  // Handle null and undefined
  if (val1 == null || val2 == null) return val1 === val2;

  // Handle different types
  if (typeof val1 !== typeof val2) return false;

  // Handle arrays
  if (Array.isArray(val1) && Array.isArray(val2)) {
    if (val1.length !== val2.length) return false;
    return val1.every((item, index) => isEqual(item, val2[index]));
  }

  // Handle objects
  if (typeof val1 === 'object' && typeof val2 === 'object') {
    const keys1 = Object.keys(val1);
    const keys2 = Object.keys(val2);

    if (keys1.length !== keys2.length) return false;

    return keys1.every((key) =>
      isEqual(val1[key as keyof typeof val1], val2[key as keyof typeof val2]),
    );
  }

  // For other types (functions, symbols, etc.)
  return false;
}
