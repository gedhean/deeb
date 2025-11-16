import { describe, it, expect } from 'vitest';
import { objDiff, objDiffExcludeUndefined } from './utils';

describe('objDiff', () => {
  describe('Basic comparisons', () => {
    it('should return empty object when objects are identical', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 1, b: 2, c: 3 };
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should return differences with values from second object', () => {
      const obj1 = { a: 1, b: 2 };
      const obj2 = { a: 1, b: 3 };
      expect(objDiff(obj1, obj2)).toEqual({ b: 3 });
    });

    it('should return all differences when multiple properties differ', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 1, b: 4, c: 5 };
      expect(objDiff(obj1, obj2)).toEqual({ b: 4, c: 5 });
    });
  });

  describe('Different attribute counts', () => {
    it('should include new properties from obj2', () => {
      const obj1 = { a: 1, b: 2 };
      const obj2 = { a: 1, b: 2, c: 3 };
      expect(objDiff(obj1, obj2)).toEqual({ c: 3 });
    });

    it('should include undefined for properties removed in obj2', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 1, b: 2 };
      expect(objDiff(obj1, obj2)).toEqual({ c: undefined });
    });

    it('should handle both added and removed properties', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 1, b: 4, d: 5 };
      expect(objDiff(obj1, obj2)).toEqual({ b: 4, c: undefined, d: 5 });
    });

    it('should handle completely different keys', () => {
      const obj1 = { a: 1, b: 2 };
      const obj2 = { c: 3, d: 4 };
      expect(objDiff(obj1, obj2)).toEqual({
        a: undefined,
        b: undefined,
        c: 3,
        d: 4,
      });
    });
  });

  describe('Nested objects', () => {
    it('should detect differences in nested objects', () => {
      const obj1 = { a: 1, nested: { x: 1, y: 2 } };
      const obj2 = { a: 1, nested: { x: 1, y: 3 } };
      expect(objDiff(obj1, obj2)).toEqual({ nested: { x: 1, y: 3 } });
    });

    it('should detect same nested objects', () => {
      const obj1 = { a: 1, nested: { x: 1, y: 2 } };
      const obj2 = { a: 1, nested: { x: 1, y: 2 } };
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should detect deeply nested differences', () => {
      const obj1 = { a: { b: { c: 1 } } };
      const obj2 = { a: { b: { c: 2 } } };
      expect(objDiff(obj1, obj2)).toEqual({ a: { b: { c: 2 } } });
    });
  });

  describe('Arrays', () => {
    it('should detect array differences', () => {
      const obj1 = { arr: [1, 2, 3] };
      const obj2 = { arr: [1, 2, 4] };
      expect(objDiff(obj1, obj2)).toEqual({ arr: [1, 2, 4] });
    });

    it('should detect same arrays', () => {
      const obj1 = { arr: [1, 2, 3] };
      const obj2 = { arr: [1, 2, 3] };
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should detect array length differences', () => {
      const obj1 = { arr: [1, 2, 3] };
      const obj2 = { arr: [1, 2] };
      expect(objDiff(obj1, obj2)).toEqual({ arr: [1, 2] });
    });

    it('should detect nested arrays with objects', () => {
      const obj1 = { arr: [{ id: 1 }, { id: 2 }] };
      const obj2 = { arr: [{ id: 1 }, { id: 2 }] };
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should detect differences in nested arrays with objects', () => {
      const obj1 = { arr: [{ id: 1 }, { id: 2 }] };
      const obj2 = { arr: [{ id: 1 }, { id: 3 }] };
      expect(objDiff(obj1, obj2)).toEqual({ arr: [{ id: 1 }, { id: 3 }] });
    });
  });

  describe('Edge cases', () => {
    it('should handle null values', () => {
      const obj1 = { a: 1, b: null };
      const obj2 = { a: 1, b: null };
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should detect null to value change', () => {
      const obj1 = { a: null };
      const obj2 = { a: 1 };
      expect(objDiff(obj1, obj2)).toEqual({ a: 1 });
    });

    it('should detect value to null change', () => {
      const obj1 = { a: 1 };
      const obj2 = { a: null };
      expect(objDiff(obj1, obj2)).toEqual({ a: null });
    });

    it('should handle undefined values', () => {
      const obj1 = { a: undefined };
      const obj2 = { a: undefined };
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should detect undefined to value change', () => {
      const obj1 = { a: undefined };
      const obj2 = { a: 1 };
      expect(objDiff(obj1, obj2)).toEqual({ a: 1 });
    });

    it('should handle empty objects', () => {
      const obj1 = {};
      const obj2 = {};
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should handle empty object vs object with properties', () => {
      const obj1 = {};
      const obj2 = { a: 1 };
      expect(objDiff(obj1, obj2)).toEqual({ a: 1 });
    });

    it('should handle boolean values', () => {
      const obj1 = { a: true, b: false };
      const obj2 = { a: true, b: true };
      expect(objDiff(obj1, obj2)).toEqual({ b: true });
    });

    it('should handle string values', () => {
      const obj1 = { name: 'John', age: '30' };
      const obj2 = { name: 'Jane', age: '30' };
      expect(objDiff(obj1, obj2)).toEqual({ name: 'Jane' });
    });

    it('should handle number 0 correctly', () => {
      const obj1 = { a: 0 };
      const obj2 = { a: 0 };
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should detect changes from 0', () => {
      const obj1 = { a: 0 };
      const obj2 = { a: 1 };
      expect(objDiff(obj1, obj2)).toEqual({ a: 1 });
    });

    it('should handle empty strings', () => {
      const obj1 = { a: '' };
      const obj2 = { a: '' };
      expect(objDiff(obj1, obj2)).toEqual({});
    });

    it('should detect empty string changes', () => {
      const obj1 = { a: '' };
      const obj2 = { a: 'value' };
      expect(objDiff(obj1, obj2)).toEqual({ a: 'value' });
    });
  });

  describe('Type coercion', () => {
    it('should treat string and number as different', () => {
      const obj1 = { a: '1' };
      const obj2 = { a: 1 };
      expect(objDiff(obj1, obj2)).toEqual({ a: 1 });
    });

    it('should treat null and undefined as different', () => {
      const obj1 = { a: null };
      const obj2 = { a: undefined };
      expect(objDiff(obj1, obj2)).toEqual({ a: undefined });
    });

    it('should treat 0 and false as different', () => {
      const obj1 = { a: 0 };
      const obj2 = { a: false };
      expect(objDiff(obj1, obj2)).toEqual({ a: false });
    });
  });
});

describe('objDiffExcludeUndefined', () => {
  describe('Basic comparisons', () => {
    it('should return empty object when objects are identical', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 1, b: 2, c: 3 };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({});
    });

    it('should return differences with values from second object', () => {
      const obj1 = { a: 1, b: 2 };
      const obj2 = { a: 1, b: 3 };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({ b: 3 });
    });

    it('should exclude undefined values from result', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 1, b: 2 };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({});
    });
  });

  describe('Different attribute counts', () => {
    it('should include new properties from obj2', () => {
      const obj1 = { a: 1, b: 2 };
      const obj2 = { a: 1, b: 2, c: 3 };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({ c: 3 });
    });

    it('should NOT include undefined for properties removed in obj2', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 1, b: 2 };
      // Unlike objDiff which returns { c: undefined }, this excludes c
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({});
    });

    it('should handle both added and removed properties', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 1, b: 4, d: 5 };
      // c: undefined is excluded
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({ b: 4, d: 5 });
    });

    it('should handle completely different keys, excluding undefined', () => {
      const obj1 = { a: 1, b: 2 };
      const obj2 = { c: 3, d: 4 };
      // a and b would be undefined, so they're excluded
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({ c: 3, d: 4 });
    });
  });

  describe('Explicit undefined values', () => {
    it('should exclude explicitly set undefined values', () => {
      const obj1 = { a: 1, b: 2 };
      const obj2 = { a: 1, b: undefined };
      // b changed from 2 to undefined, but undefined is excluded
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({});
    });

    it('should handle changing from undefined to a value', () => {
      const obj1 = { a: undefined };
      const obj2 = { a: 1 };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({ a: 1 });
    });

    it('should exclude if both are undefined', () => {
      const obj1 = { a: undefined };
      const obj2 = { a: undefined };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({});
    });
  });

  describe('Complex scenarios', () => {
    it('should handle mixed changes with some undefined', () => {
      const obj1 = { a: 1, b: 2, c: 3, d: 4 };
      const obj2 = { a: 1, b: 5, e: 6 };
      // c and d would be undefined, so excluded. b changed, e is new
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({ b: 5, e: 6 });
    });

    it('should keep null values but exclude undefined', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: null, b: 2 };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({ a: null });
    });
  });

  describe('Arrays and nested objects', () => {
    it('should handle arrays correctly', () => {
      const obj1 = { arr: [1, 2, 3] };
      const obj2 = { arr: [1, 2, 4] };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({ arr: [1, 2, 4] });
    });

    it('should handle nested objects correctly', () => {
      const obj1 = { a: 1, nested: { x: 1, y: 2 } };
      const obj2 = { a: 1, nested: { x: 1, y: 3 } };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({
        nested: { x: 1, y: 3 },
      });
    });
  });

  describe('Edge cases', () => {
    it('should handle empty objects', () => {
      const obj1 = {};
      const obj2 = {};
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({});
    });

    it('should handle all values being undefined', () => {
      const obj1 = { a: 1, b: 2 };
      const obj2 = {};
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({});
    });

    it('should handle falsy values that are not undefined', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { a: 0, b: false, c: '' };
      expect(objDiffExcludeUndefined(obj1, obj2)).toEqual({
        a: 0,
        b: false,
        c: '',
      });
    });
  });
});
