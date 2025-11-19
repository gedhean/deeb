/**
 * DataTableFacetedFilter Component
 *
 * A multi-select filter with faceted counts for filtering table data.
 * Shows the count of items for each option using TanStack Table's faceting API.
 *
 * @example
 * ```tsx
 * <DataTableFacetedFilter
 *   column={table.getColumn('role')}
 *   title="Role"
 *   options={[
 *     { label: 'Admin', value: 'admin', icon: Shield },
 *     { label: 'User', value: 'user', icon: User },
 *   ]}
 * />
 * ```
 */

import { Check, PlusCircle, X } from 'lucide-react';

import { cn } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Separator } from '@/components/ui/separator';
import type { SelectFilterProps } from './types';

export interface DataTableFacetedFilterProps<TData> extends SelectFilterProps<TData> {
  /** If true, shows an X button to remove the filter from toolbar */
  showRemove?: boolean;
  /** Callback when the remove button is clicked */
  onRemove?: () => void;
}

export function DataTableFacetedFilter<TData>({
  column,
  title,
  options,
  showRemove = false,
  onRemove,
}: DataTableFacetedFilterProps<TData>) {
  if (!column) {
    return null;
  }

  const facets = column.getFacetedUniqueValues();
  const rawFilter = column.getFilterValue();
  const values = Array.isArray(rawFilter) ? rawFilter : [];
  const selectedValues = new Set(values as string[]);
  const hasValue = selectedValues.size > 0;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          <PlusCircle className="mr-2 h-4 w-4" />
          {title}
          {hasValue && (
            <>
              <Separator orientation="vertical" className="mx-2 h-4" />
              <Badge variant="secondary" className="rounded-sm px-1 font-normal lg:hidden">
                {selectedValues.size}
              </Badge>
              <div className="hidden space-x-1 lg:flex">
                {selectedValues.size > 2 ? (
                  <Badge variant="secondary" className="rounded-sm px-1 font-normal">
                    {selectedValues.size} selected
                  </Badge>
                ) : (
                  options
                    .filter((option) => selectedValues.has(option.value))
                    .map((option) => (
                      <Badge
                        variant="secondary"
                        key={option.value}
                        className="rounded-sm px-1 font-normal"
                      >
                        {option.label}
                      </Badge>
                    ))
                )}
              </div>
            </>
          )}
          {showRemove && hasValue && onRemove && (
            <>
              <Separator orientation="vertical" className="mx-2 h-4" />
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
                aria-label={`Remove ${title} filter`}
                className="flex items-center hover:text-foreground text-muted-foreground"
              >
                <X className="h-3 w-3" />
              </div>
            </>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0" align="start">
        <Command>
          <CommandInput placeholder={title} />
          <CommandList>
            <CommandEmpty>No results found.</CommandEmpty>
            <CommandGroup>
              {options.map((option) => {
                const isSelected = selectedValues.has(option.value);
                return (
                  <CommandItem
                    key={option.value}
                    onSelect={() => {
                      if (isSelected) {
                        selectedValues.delete(option.value);
                      } else {
                        selectedValues.add(option.value);
                      }
                      const filterValues = Array.from(selectedValues);
                      column?.setFilterValue(filterValues.length ? filterValues : undefined);
                    }}
                  >
                    <div
                      className={cn(
                        'mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary',
                        isSelected
                          ? 'bg-primary text-primary-foreground'
                          : 'opacity-50 [&_svg]:invisible',
                      )}
                    >
                      <Check className={cn('h-4 w-4')} />
                    </div>
                    {option.icon && <option.icon className="mr-2 h-4 w-4 text-muted-foreground" />}
                    <span>{option.label}</span>
                    {facets?.get(option.value) && (
                      <span className="ml-auto flex h-4 w-4 items-center justify-center font-mono text-xs">
                        {facets.get(option.value)}
                      </span>
                    )}
                  </CommandItem>
                );
              })}
            </CommandGroup>
            {selectedValues.size > 0 && (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem
                    onSelect={() => column?.setFilterValue(undefined)}
                    className="justify-center text-center"
                  >
                    Clear filters
                  </CommandItem>
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
