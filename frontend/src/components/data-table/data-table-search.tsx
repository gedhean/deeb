/**
 * DataTableSearch Component
 *
 * A text search input for filtering table columns.
 * Supports debounced input for performance optimization.
 *
 * @example
 * ```tsx
 * <DataTableSearch
 *   column={table.getColumn('email')}
 *   placeholder="Search emails..."
 * />
 * ```
 */

import * as React from 'react';
import { Search, X } from 'lucide-react';

import { Input } from '@/components/ui/input';
import type { SearchFilterProps } from './types';

export interface DataTableSearchProps<TData> extends SearchFilterProps<TData> {
  /** If true, shows an X button to remove the filter from toolbar */
  showRemove?: boolean;
  /** Callback when the remove button is clicked */
  onRemove?: () => void;
}

export function DataTableSearch<TData>({
  column,
  placeholder,
  showRemove = false,
  onRemove,
}: DataTableSearchProps<TData>) {
  const [value, setValue] = React.useState<string>((column?.getFilterValue() as string) ?? '');

  // Sync local state with column filter value (for reset functionality)
  const filterValue = (column?.getFilterValue() as string) ?? '';
  React.useEffect(() => {
    setValue(filterValue);
  }, [filterValue]);

  // Debounce filter updates for performance
  React.useEffect(() => {
    const timeout = setTimeout(() => {
      column?.setFilterValue(value);
    }, 300);

    return () => clearTimeout(timeout);
  }, [column, value]);

  if (!column) {
    return null;
  }

  const hasValue = value !== '';

  return (
    <div className="relative">
      <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        placeholder={placeholder ?? 'Search...'}
        value={value}
        onChange={(event) => setValue(event.target.value)}
        className="h-8 w-[150px] pl-8 pr-8 lg:w-[250px]"
      />
      {showRemove && hasValue && onRemove && (
        <div
          role="button"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.stopPropagation();
              e.preventDefault();
              onRemove();
            }
          }}
          tabIndex={0}
          aria-label="Remove search filter"
          className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center hover:text-foreground text-muted-foreground cursor-pointer"
        >
          <X className="h-3 w-3" />
        </div>
      )}
    </div>
  );
}
