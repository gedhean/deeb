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
  /** Type of filter component to render */
  type: 'search' | 'faceted' | 'select' | 'dateRange';
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
}
