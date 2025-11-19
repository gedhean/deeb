/**
 * DataTableSelectFilter Component
 *
 * A single-select dropdown filter for filtering table data.
 * Shows faceted counts for each option.
 *
 * @example
 * ```tsx
 * <DataTableSelectFilter
 *   column={table.getColumn('status')}
 *   title="Status"
 *   options={[
 *     { label: 'Active', value: 'active' },
 *     { label: 'Inactive', value: 'inactive' },
 *   ]}
 * />
 * ```
 */

import { Check, ChevronDown, X } from 'lucide-react';

import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import type { SelectFilterProps } from './types';

export interface DataTableSelectFilterProps<TData> extends SelectFilterProps<TData> {
  /** If true, shows an X button to remove the filter from toolbar */
  showRemove?: boolean;
  /** Callback when the remove button is clicked */
  onRemove?: () => void;
}

export function DataTableSelectFilter<TData>({
  column,
  title,
  options,
  showRemove = false,
  onRemove,
}: DataTableSelectFilterProps<TData>) {
  const facets = column?.getFacetedUniqueValues();
  const selectedValue = column?.getFilterValue() as string | undefined;

  if (!column) {
    return null;
  }

  const selectedOption = options.find((option) => option.value === selectedValue);
  const hasValue = selectedValue !== undefined;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          {selectedOption?.icon && <selectedOption.icon className="mr-2 h-4 w-4" />}
          {selectedOption?.label ?? title}
          {!hasValue && <ChevronDown className="ml-2 h-4 w-4 text-muted-foreground" />}
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
                const isSelected = selectedValue === option.value;
                return (
                  <CommandItem
                    key={option.value}
                    onSelect={() => {
                      column?.setFilterValue(isSelected ? undefined : option.value);
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
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
