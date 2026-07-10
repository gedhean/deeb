/**
 * DataTableAsyncFilter Component
 *
 * A single-select filter whose options are fetched from the server as the user
 * types (debounced), instead of being preloaded. Use for reference-backed
 * filters (database instance, project, environment, author, …) that can have
 * more entries than is practical to load into a dropdown.
 *
 * The column filter value is the selected option's `value` (an id). The control
 * resolves the display label for the current value — including one loaded from
 * the URL — via `resolveLabel`.
 *
 * @example
 * <DataTableAsyncFilter
 *   column={table.getColumn('instance')}
 *   title="Instance"
 *   loadOptions={(q) => instancesService.list({ search: q, limit: 20 })
 *     .then((r) => r.items.map((i) => ({ label: i.name, value: i.id })))}
 *   resolveLabel={(id) => instancesService.get(id).then((i) => i.name)}
 * />
 */

import * as React from 'react';
import type { Column } from '@tanstack/react-table';
import { Check, ChevronDown, Loader2, X } from 'lucide-react';

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
import { useDebounce } from '@/hooks/useDebounce';
import type { FilterOption } from './types';

export interface DataTableAsyncFilterProps<TData> {
  column: Column<TData, unknown>;
  title: string;
  placeholder?: string;
  /** Fetch matching options for the typed term (should be bounded, e.g. limit 20). */
  loadOptions: (search: string) => Promise<FilterOption[]>;
  /** Resolve the label for an already-selected value (e.g. an id from the URL). */
  resolveLabel?: (value: string) => Promise<string | undefined>;
  /** If true, shows an X button to remove the filter from the toolbar. */
  showRemove?: boolean;
  onRemove?: () => void;
}

export function DataTableAsyncFilter<TData>({
  column,
  title,
  placeholder,
  loadOptions,
  resolveLabel,
  showRemove = false,
  onRemove,
}: DataTableAsyncFilterProps<TData>) {
  const selectedValue = column?.getFilterValue() as string | undefined;

  const [open, setOpen] = React.useState(false);
  const [search, setSearch] = React.useState('');
  const debouncedSearch = useDebounce(search, 300);
  const [options, setOptions] = React.useState<FilterOption[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [selected, setSelected] = React.useState<FilterOption | null>(null);

  // Keep the displayed label in sync with the current value, resolving it when
  // the value was set externally (e.g. loaded from the URL) and we don't yet
  // have its label.
  React.useEffect(() => {
    if (!selectedValue) {
      setSelected(null);
      return;
    }
    if (selected?.value === selectedValue) {
      return;
    }
    let active = true;
    if (resolveLabel) {
      resolveLabel(selectedValue).then((label) => {
        if (active) {
          setSelected({ value: selectedValue, label: label ?? selectedValue });
        }
      });
    } else {
      setSelected({ value: selectedValue, label: selectedValue });
    }
    return () => {
      active = false;
    };
  }, [selectedValue, selected?.value, resolveLabel]);

  // Fetch options while the popover is open, re-running on debounced input.
  React.useEffect(() => {
    if (!open) {
      return;
    }
    let active = true;
    setLoading(true);
    loadOptions(debouncedSearch)
      .then((opts) => {
        if (active) {
          setOptions(opts);
        }
      })
      .catch(() => {
        if (active) {
          setOptions([]);
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [open, debouncedSearch, loadOptions]);

  if (!column) {
    return null;
  }

  const hasValue = selectedValue !== undefined && selectedValue !== '';

  const clear = () => {
    column.setFilterValue(undefined);
    setSelected(null);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          {hasValue ? (
            <>
              <span className="text-muted-foreground">{title}:</span>
              <span className="ml-1">{selected?.label ?? selectedValue}</span>
            </>
          ) : (
            <>
              {title}
              <ChevronDown className="ml-2 h-4 w-4 text-muted-foreground" />
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
      <PopoverContent className="w-[250px] p-0" align="start">
        {/* shouldFilter=false: the server already filtered by the typed term. */}
        <Command shouldFilter={false}>
          <CommandInput
            placeholder={placeholder ?? `Search ${title.toLowerCase()}...`}
            value={search}
            onValueChange={setSearch}
          />
          <CommandList>
            {loading ? (
              <div className="flex items-center justify-center py-6 text-sm text-muted-foreground">
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Searching...
              </div>
            ) : (
              <>
                <CommandEmpty>No results found.</CommandEmpty>
                <CommandGroup>
                  {options.map((option) => {
                    const isSelected = selectedValue === option.value;
                    return (
                      <CommandItem
                        key={option.value}
                        value={option.value}
                        onSelect={() => {
                          if (isSelected) {
                            clear();
                          } else {
                            column.setFilterValue(option.value);
                            setSelected(option);
                          }
                          setOpen(false);
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
                          <Check className="h-4 w-4" />
                        </div>
                        <span>{option.label}</span>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
