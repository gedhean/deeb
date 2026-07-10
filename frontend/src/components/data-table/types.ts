/**
 * Data Table Types
 *
 * Shared types and interfaces for data table components including
 * toolbar, filters, and column configurations.
 */

import type { LucideIcon } from 'lucide-react';
import type { Column } from '@tanstack/react-table';

/**
 * Supported filter variants for data table columns
 */
export type FilterVariant = 'search' | 'select' | 'multi-select' | 'date-range';

/**
 * Filter option for select and multi-select filters
 */
export interface FilterOption {
  /** Display label for the option */
  label: string;
  /** Value to filter by */
  value: string;
  /** Optional icon to display next to the label */
  icon?: LucideIcon;
  /** Optional count for faceted filtering (auto-populated) */
  count?: number;
}

/**
 * Filter configuration stored in column metadata
 */
export interface FilterConfig {
  /** Type of filter to render */
  variant: FilterVariant;
  /** Title to display in the filter UI */
  title?: string;
  /** Placeholder text for search filters */
  placeholder?: string;
  /** Options for select/multi-select filters */
  options?: FilterOption[];
}

/**
 * Extended column metadata with filter configuration
 */
export interface ColumnMeta {
  /** Filter configuration for this column */
  filterConfig?: FilterConfig;
  /** Column display name */
  displayName?: string;
}

/**
 * Props for filter components
 */
export interface BaseFilterProps<TData> {
  /** TanStack Table column instance */
  column: Column<TData, unknown>;
  /** Title to display in the filter UI */
  title: string;
}

export interface SearchFilterProps<TData> extends Omit<BaseFilterProps<TData>, 'title'> {
  /** Placeholder text for search input */
  placeholder?: string;
}

export interface SelectFilterProps<TData> extends BaseFilterProps<TData> {
  /** Options for the filter */
  options: FilterOption[];
}

export interface DateRangeFilterProps<TData> extends BaseFilterProps<TData> {
  /** Optional preset date ranges */
  presets?: Array<{
    label: string;
    getValue: () => [Date, Date];
  }>;
}

/**
 * Date range preset configuration
 */
export interface DateRangePreset {
  label: string;
  getValue: () => [Date, Date];
}

/**
 * Filter definition for dynamic toolbar filters
 * Used to define available filters that can be added/removed from the toolbar
 */
export interface FilterDefinition {
  /** Column ID that this filter targets */
  id: string;
  /**
   * Type of filter component to render.
   * - 'search'    : free-text input (single string value)
   * - 'faceted'   : multi-select over a static option list (string[] value)
   * - 'select'    : single-select over a static option list (string value)
   * - 'async'     : single-select whose options are fetched from the server as
   *                 you type (string value = selected id); scales past a preload
   * - 'dateRange' : date range picker ([Date, Date] value)
   */
  type: 'search' | 'faceted' | 'select' | 'async' | 'dateRange';
  /** Title to display in the filter UI */
  title: string;
  /** Placeholder text for search filters */
  placeholder?: string;
  /** Options for select/faceted filters */
  options?: FilterOption[];
  /** Preset date ranges for date range filters */
  presets?: DateRangePreset[];
  /** If true, filter cannot be removed from toolbar */
  pinned?: boolean;
  /** If true, filter is visible by default */
  defaultActive?: boolean;
  /**
   * For type 'async': fetch matching options for the typed search term from the
   * server (debounced). Should return a bounded list (e.g. limit 20).
   */
  loadOptions?: (search: string) => Promise<FilterOption[]>;
  /**
   * For type 'async': resolve the display label for an already-selected value
   * (e.g. an id loaded from the URL) so the control can show its name.
   */
  resolveLabel?: (value: string) => Promise<string | undefined>;
}
