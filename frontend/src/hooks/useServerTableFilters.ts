/**
 * useServerTableFilters
 *
 * Bridges the DataTable/TanStack column-filter state to server-side query
 * params, and persists the full view (search, filters, page, page size) in the
 * URL so it is shareable and survives refresh / back-forward.
 *
 * The URL mirrors the API query string, e.g.
 *   ?search=pay&status=Active&database_instance_id=<id>&created_after=...&page=2&pageSize=50
 * (`page` is 1-based; `pageSize` is always present once interacted with).
 *
 * Pages using this must:
 *  - pass `filterDefinitions` (so the hook knows each filter's value shape),
 *  - set `pagination.manualFiltering: true` on <DataTable>,
 *  - pass the returned `onFiltersChange` and `initialColumnFilters`.
 */

import { useCallback, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import type { ColumnFiltersState } from '@tanstack/react-table';
import type { FilterDefinition } from '@/components/data-table/types';

/** Describes how table column-filter ids map to API/URL query params. */
export interface ServerTableFilterMapping {
  /** Column ids whose (string) value maps to the shared `search` param. */
  searchColumnIds?: string[];
  /** Column id -> param name for equality filters (faceted/select/async). */
  equals?: Record<string, string>;
  /** Column id -> [afterParam, beforeParam] for date-range filters. */
  dateRanges?: Record<string, [string, string]>;
}

export interface UseServerTableFiltersOptions {
  mapping: ServerTableFilterMapping;
  /** The page's filter definitions — used to know each column's value shape. */
  filterDefinitions?: FilterDefinition[];
  /** Initial page size when the URL doesn't specify one (default 50). */
  initialPageSize?: number;
  /** Default column filters applied when the URL carries no filter state. */
  initialColumnFilters?: ColumnFiltersState;
}

export type ServerTableFilterParams = Record<string, string | number | undefined>;

export interface UseServerTableFiltersResult {
  /** Flat params (limit/offset + active filters) to pass to the list data hook. */
  params: ServerTableFilterParams;
  pageIndex: number;
  pageSize: number;
  setPageIndex: (pageIndex: number) => void;
  setPageSize: (pageSize: number) => void;
  /** Effective initial column filters (URL state if present, else defaults). */
  initialColumnFilters: ColumnFiltersState;
  /** Pass to <DataTable onFiltersChange={...}>. Resets to the first page on change. */
  onFiltersChange: (filters: ColumnFiltersState) => void;
}

/** Take the first meaningful value from a faceted (array) or single filter value. */
function firstValue(value: unknown): string | undefined {
  const raw = Array.isArray(value) ? value[0] : value;
  if (raw === undefined || raw === null || raw === '') {
    return undefined;
  }
  return String(raw);
}

function toParams(
  filters: ColumnFiltersState,
  mapping: ServerTableFilterMapping,
): ServerTableFilterParams {
  const params: ServerTableFilterParams = {};
  const searchIds = new Set(mapping.searchColumnIds ?? []);

  for (const filter of filters) {
    const { id, value } = filter;

    if (searchIds.has(id)) {
      if (typeof value === 'string' && value !== '') {
        params.search = value;
      }
      continue;
    }

    const dateRange = mapping.dateRanges?.[id];
    if (dateRange) {
      const [afterParam, beforeParam] = dateRange;
      const range = value as [Date | string | undefined, Date | string | undefined] | undefined;
      if (Array.isArray(range)) {
        if (range[0]) {
          params[afterParam] = new Date(range[0]).toISOString();
        }
        if (range[1]) {
          params[beforeParam] = new Date(range[1]).toISOString();
        }
      }
      continue;
    }

    const equalsParam = mapping.equals?.[id];
    if (equalsParam) {
      if (Array.isArray(value)) {
        // Faceted (multi-select): send every selected value, comma-joined -> IN (...).
        const vals = value.filter((v) => v !== undefined && v !== null && v !== '').map(String);
        if (vals.length > 0) {
          params[equalsParam] = vals.join(',');
        }
      } else {
        // Async/select (single value).
        const v = firstValue(value);
        if (v !== undefined) {
          params[equalsParam] = v;
        }
      }
    }
  }

  return params;
}

/** All URL/query param names this hook manages, for gating and cleanup. */
function managedParamNames(mapping: ServerTableFilterMapping): string[] {
  const names = new Set<string>(['page', 'pageSize']);
  if ((mapping.searchColumnIds ?? []).length > 0) {
    names.add('search');
  }
  Object.values(mapping.equals ?? {}).forEach((p) => names.add(p));
  Object.values(mapping.dateRanges ?? {}).forEach(([a, b]) => {
    names.add(a);
    names.add(b);
  });
  return Array.from(names);
}

/** Rebuild column-filter state from the URL query params (reverse of toParams). */
function urlToColumnFilters(
  search: URLSearchParams,
  mapping: ServerTableFilterMapping,
  facetedIds: Set<string>,
): ColumnFiltersState {
  const filters: ColumnFiltersState = [];

  const searchColumnId = (mapping.searchColumnIds ?? [])[0];
  if (searchColumnId) {
    const v = search.get('search');
    if (v) {
      filters.push({ id: searchColumnId, value: v });
    }
  }

  Object.entries(mapping.equals ?? {}).forEach(([columnId, param]) => {
    const v = search.get(param);
    if (v) {
      // Faceted filters carry an array (a comma-joined list in the URL); async/
      // select filters carry a single string.
      filters.push({ id: columnId, value: facetedIds.has(columnId) ? v.split(',') : v });
    }
  });

  Object.entries(mapping.dateRanges ?? {}).forEach(([columnId, [afterParam, beforeParam]]) => {
    const after = search.get(afterParam);
    const before = search.get(beforeParam);
    if (after || before) {
      filters.push({
        id: columnId,
        value: [after ? new Date(after) : undefined, before ? new Date(before) : undefined],
      });
    }
  });

  return filters;
}

export function useServerTableFilters({
  mapping,
  filterDefinitions,
  initialPageSize = 50,
  initialColumnFilters = [],
}: UseServerTableFiltersOptions): UseServerTableFiltersResult {
  const [searchParams, setSearchParams] = useSearchParams();

  const facetedIds = useMemo(
    () => new Set((filterDefinitions ?? []).filter((f) => f.type === 'faceted').map((f) => f.id)),
    [filterDefinitions],
  );
  const managedNames = useMemo(() => managedParamNames(mapping), [mapping]);

  // Derive the initial state from the URL once, on mount. If the URL carries no
  // managed params, fall back to the page's default filters / page size.
  // The URL is the source of truth thereafter, updated via the setters below.
  const initialSearchParamsRef = useRef(searchParams);
  const initial = useMemo(() => {
    const search = initialSearchParamsRef.current;
    const hasUrlState = managedNames.some((n) => search.has(n));
    const columnFilters = hasUrlState
      ? urlToColumnFilters(search, mapping, facetedIds)
      : initialColumnFilters;
    const pageParam = Number(search.get('page'));
    const sizeParam = Number(search.get('pageSize'));
    return {
      columnFilters,
      pageIndex: Number.isFinite(pageParam) && pageParam > 0 ? pageParam - 1 : 0,
      pageSize: Number.isFinite(sizeParam) && sizeParam > 0 ? sizeParam : initialPageSize,
    };
    // Run once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [pageIndex, setPageIndexState] = useState(initial.pageIndex);
  const [pageSize, setPageSizeState] = useState(initial.pageSize);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initial.columnFilters);

  // Refs so the sync callbacks stay identity-stable (avoids re-triggering the
  // DataTable's onFiltersChange effect on every render).
  const stateRef = useRef({ columnFilters, pageIndex, pageSize });
  stateRef.current = { columnFilters, pageIndex, pageSize };

  const syncUrl = useCallback(
    (cf: ColumnFiltersState, pi: number, ps: number) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          managedNames.forEach((n) => next.delete(n));
          const filterParams = toParams(cf, mapping);
          Object.entries(filterParams).forEach(([k, v]) => {
            if (v !== undefined && v !== '') {
              next.set(k, String(v));
            }
          });
          if (pi > 0) {
            next.set('page', String(pi + 1));
          }
          next.set('pageSize', String(ps));
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams, managedNames, mapping],
  );

  const onFiltersChange = useCallback(
    (filters: ColumnFiltersState) => {
      const prev = stateRef.current.columnFilters;
      if (JSON.stringify(prev) === JSON.stringify(filters)) {
        return;
      }
      setColumnFilters(filters);
      setPageIndexState(0);
      syncUrl(filters, 0, stateRef.current.pageSize);
    },
    [syncUrl],
  );

  const setPageIndex = useCallback(
    (pi: number) => {
      setPageIndexState(pi);
      syncUrl(stateRef.current.columnFilters, pi, stateRef.current.pageSize);
    },
    [syncUrl],
  );

  const setPageSize = useCallback(
    (ps: number) => {
      setPageSizeState(ps);
      setPageIndexState(0);
      syncUrl(stateRef.current.columnFilters, 0, ps);
    },
    [syncUrl],
  );

  const filterParams = useMemo(() => toParams(columnFilters, mapping), [columnFilters, mapping]);

  const params = useMemo<ServerTableFilterParams>(
    () => ({
      limit: pageSize,
      offset: pageIndex * pageSize,
      ...filterParams,
    }),
    [pageSize, pageIndex, filterParams],
  );

  return {
    params,
    pageIndex,
    pageSize,
    setPageIndex,
    setPageSize,
    initialColumnFilters: initial.columnFilters,
    onFiltersChange,
  };
}
