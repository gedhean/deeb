/**
 * DataTableFilterButton Component
 *
 * A button that opens a popover showing all available filters that can be added to the toolbar.
 * Displays a checkmark next to filters that are currently active in the toolbar.
 *
 * @example
 * ```tsx
 * <DataTableFilterButton
 *   filters={filterDefinitions}
 *   activeFilterIds={new Set(['status', 'project'])}
 *   onToggleFilter={(id) => handleToggle(id)}
 *   table={table}
 * />
 * ```
 */

import * as React from 'react';
import { Check, ListFilter } from 'lucide-react';
import type { Table } from '@tanstack/react-table';

import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import type { FilterDefinition } from './types';

export interface DataTableFilterButtonProps<TData> {
  /** Array of all available filter definitions */
  filters: FilterDefinition[];
  /** Set of currently active filter IDs */
  activeFilterIds: Set<string>;
  /** Callback when a filter is toggled */
  onToggleFilter: (filterId: string) => void;
  /** TanStack Table instance (for future enhancements) */
  table: Table<TData>;
}

export function DataTableFilterButton<TData>({
  filters,
  activeFilterIds,
  onToggleFilter,
}: DataTableFilterButtonProps<TData>) {
  const [open, setOpen] = React.useState(false);

  // Display order + headings for the known filter types.
  const typeOrder = ['search', 'faceted', 'select', 'async', 'dateRange'];
  const typeLabels: Record<string, string> = {
    search: 'Search',
    faceted: 'Multi-select',
    select: 'Select',
    async: 'Lookup',
    dateRange: 'Date Range',
  };

  // Group filters by type. Build the map from the filters themselves so any
  // type (including 'async') is included rather than silently dropped.
  const filtersByType = React.useMemo(() => {
    const grouped: Record<string, FilterDefinition[]> = {};
    filters.forEach((filter) => {
      (grouped[filter.type] ??= []).push(filter);
    });
    return grouped;
  }, [filters]);

  const orderedTypes = React.useMemo(() => {
    const present = Object.keys(filtersByType);
    const known = typeOrder.filter((t) => present.includes(t));
    const extra = present.filter((t) => !typeOrder.includes(t));
    return [...known, ...extra];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filtersByType]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          <ListFilter className="mr-2 h-4 w-4" />
          Filter
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0" align="start">
        <Command>
          <CommandInput placeholder="Search filters..." />
          <CommandList>
            <CommandEmpty>No filters found.</CommandEmpty>
            {orderedTypes.map((type) => {
              const typeFilters = filtersByType[type];
              if (!typeFilters || typeFilters.length === 0) return null;

              return (
                <CommandGroup key={type} heading={typeLabels[type] ?? type}>
                  {typeFilters.map((filter) => {
                    const isActive = activeFilterIds.has(filter.id);
                    const isPinned = filter.pinned;

                    return (
                      <CommandItem
                        key={filter.id}
                        onSelect={() => {
                          // Don't allow toggling pinned filters
                          if (!isPinned) {
                            onToggleFilter(filter.id);
                            // Keep popover open for multiple selections
                          }
                        }}
                        disabled={isPinned}
                        className={isPinned ? 'opacity-50 cursor-not-allowed' : ''}
                      >
                        <div className="flex items-center justify-between w-full">
                          <span>{filter.title}</span>
                          {isActive && <Check className="h-4 w-4 text-primary" />}
                        </div>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              );
            })}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
