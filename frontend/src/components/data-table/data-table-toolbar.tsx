/**
 * DataTableToolbar Component
 *
 * A flexible toolbar container for data tables that supports:
 * - Dynamic filter visibility management
 * - Filter button to add/remove filters
 * - Search input
 * - Faceted filters (multi-select with counts)
 * - Select filters (single selection)
 * - Date range filters
 * - Column visibility toggle
 *
 * Uses FilterDefinition array to configure available filters.
 * Only renders filters that are active (have values or are marked as defaultActive).
 *
 * @example
 * ```tsx
 * <DataTable columns={columns} data={data}>
 *   {(table) => (
 *     <DataTableToolbar
 *       table={table}
 *       filters={[
 *         {
 *           id: 'email',
 *           type: 'search',
 *           placeholder: 'Search emails...',
 *           defaultActive: true,
 *         },
 *         {
 *           id: 'status',
 *           type: 'faceted',
 *           title: 'Status',
 *           options: statusOptions,
 *         },
 *       ]}
 *     >
 *       <DataTableViewOptions table={table} />
 *     </DataTableToolbar>
 *   )}
 * </DataTable>
 * ```
 */

import * as React from 'react';
import type { Table } from '@tanstack/react-table';
import { X } from 'lucide-react';

import { Button } from '@/components/ui/button';
import type { FilterDefinition } from './types';
import { DataTableFilterButton } from './data-table-filter-button';
import { DataTableSearch } from './data-table-search';
import { DataTableFacetedFilter } from './data-table-faceted-filter';
import { DataTableSelectFilter } from './data-table-select-filter';
import { DataTableAsyncFilter } from './data-table-async-filter';
import { DataTableDateRangeFilter } from './data-table-date-range-filter';

export interface DataTableToolbarProps<TData> {
  /** TanStack Table instance */
  table: Table<TData>;
  /** Filter definitions */
  filters?: FilterDefinition[];
  /** Additional toolbar content (e.g., view options) */
  children?: React.ReactNode;
}

export function DataTableToolbar<TData>({
  table,
  filters = [],
  children,
}: DataTableToolbarProps<TData>) {
  // Get table ID for localStorage key (use first column name as fallback)
  const tableId = table.getAllColumns()[0]?.id || 'default';
  const storageKey = `datatable-active-filters-${tableId}`;

  // Initialize active filters from localStorage or defaults
  const [activeFilterIds, setActiveFilterIds] = React.useState<Set<string>>(() => {
    const stored = localStorage.getItem(storageKey);
    if (stored) {
      try {
        return new Set(JSON.parse(stored));
      } catch {
        // Fallback to defaults if parsing fails
      }
    }
    // Default: pinned filters + defaultActive filters
    return new Set(filters.filter((f) => f.pinned || f.defaultActive).map((f) => f.id));
  });

  // Persist active filter IDs to localStorage
  React.useEffect(() => {
    localStorage.setItem(storageKey, JSON.stringify(Array.from(activeFilterIds)));
  }, [activeFilterIds, storageKey]);

  // Toggle filter visibility
  const handleToggleFilter = React.useCallback((filterId: string) => {
    setActiveFilterIds((prev) => {
      const next = new Set(prev);
      if (next.has(filterId)) {
        next.delete(filterId);
      } else {
        next.add(filterId);
      }
      return next;
    });
  }, []);

  // Remove filter and clear its value
  const handleRemoveFilter = React.useCallback(
    (filterId: string) => {
      // Clear the filter value
      const column = table.getColumn(filterId);
      column?.setFilterValue(undefined);

      // Remove from active filters (unless pinned)
      const filter = filters.find((f) => f.id === filterId);
      if (!filter?.pinned) {
        setActiveFilterIds((prev) => {
          const next = new Set(prev);
          next.delete(filterId);
          return next;
        });
      }
    },
    [table, filters],
  );

  const isFiltered = table.getState().columnFilters.length > 0;

  // Filter active filters: those with values OR explicitly marked active
  const activeFilters = React.useMemo(() => {
    return filters.filter((filter) => {
      // Always show pinned filters
      if (filter.pinned) return true;

      // Show if explicitly active
      if (activeFilterIds.has(filter.id)) {
        return true;
      }

      // Otherwise, only show if it has a value
      const column = table.getColumn(filter.id);
      const value = column?.getFilterValue();
      if (value !== undefined) {
        // For arrays (faceted filters), check length
        if (Array.isArray(value)) {
          return value.length > 0;
        }
        // For strings (search), check non-empty
        if (typeof value === 'string') {
          return value !== '';
        }
        // For other values, consider truthy as having value
        return true;
      }

      return false;
    });
  }, [filters, activeFilterIds, table]);

  // Render a filter component based on its type
  const renderFilter = React.useCallback(
    (filter: FilterDefinition) => {
      const column = table.getColumn(filter.id);
      if (!column) return null;

      const isPinned = filter.pinned;
      const showRemove = !isPinned;
      const onRemove = () => handleRemoveFilter(filter.id);

      switch (filter.type) {
        case 'search':
          return (
            <DataTableSearch
              key={filter.id}
              column={column}
              placeholder={filter.placeholder}
              showRemove={showRemove}
              onRemove={onRemove}
            />
          );
        case 'faceted':
          return (
            <DataTableFacetedFilter
              key={filter.id}
              column={column}
              title={filter.title}
              options={filter.options || []}
              showRemove={showRemove}
              onRemove={onRemove}
            />
          );
        case 'select':
          return (
            <DataTableSelectFilter
              key={filter.id}
              column={column}
              title={filter.title}
              options={filter.options || []}
              showRemove={showRemove}
              onRemove={onRemove}
            />
          );
        case 'async':
          if (!filter.loadOptions) return null;
          return (
            <DataTableAsyncFilter
              key={filter.id}
              column={column}
              title={filter.title}
              placeholder={filter.placeholder}
              loadOptions={filter.loadOptions}
              resolveLabel={filter.resolveLabel}
              showRemove={showRemove}
              onRemove={onRemove}
            />
          );
        case 'dateRange':
          return (
            <DataTableDateRangeFilter
              key={filter.id}
              column={column}
              title={filter.title}
              presets={filter.presets}
              showRemove={showRemove}
              onRemove={onRemove}
            />
          );
        default:
          return null;
      }
    },
    [table, handleRemoveFilter],
  );

  return (
    <div className="flex items-center justify-between" role="toolbar" aria-orientation="horizontal">
      {/* Left section: Filters and reset button */}
      <div className="flex flex-1 flex-wrap items-center gap-2">
        {/* Filter button - only show if there are filters */}
        {filters.length > 0 && (
          <DataTableFilterButton
            table={table}
            filters={filters}
            activeFilterIds={activeFilterIds}
            onToggleFilter={handleToggleFilter}
          />
        )}

        {/* Active filters */}
        {activeFilters.map((filter) => renderFilter(filter))}

        {/* Reset button */}
        {isFiltered && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.resetColumnFilters()}
            className="h-8 px-2 lg:px-3 border-dashed"
          >
            <X className="mr-2 h-4 w-4" />
            Reset
          </Button>
        )}
      </div>

      {/* Right section: Additional content (e.g., view options) */}
      {children && <div className="flex items-center gap-2">{children}</div>}
    </div>
  );
}
