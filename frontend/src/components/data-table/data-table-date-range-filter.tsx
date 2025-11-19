/**
 * DataTableDateRangeFilter Component
 *
 * A date range picker filter for filtering table data by date ranges.
 * Supports both custom date selection and preset ranges.
 *
 * @example
 * ```tsx
 * <DataTableDateRangeFilter
 *   column={table.getColumn('created_at')}
 *   title="Created"
 *   presets={[
 *     {
 *       label: 'Last 7 days',
 *       getValue: () => [subDays(new Date(), 7), new Date()],
 *     },
 *   ]}
 * />
 * ```
 */

import * as React from 'react';
import { CalendarIcon, X } from 'lucide-react';
import { format } from 'date-fns';
import type { DateRange } from 'react-day-picker';

import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import type { DateRangeFilterProps } from './types';

export interface DataTableDateRangeFilterProps<TData> extends DateRangeFilterProps<TData> {
  /** If true, shows an X button to remove the filter from toolbar */
  showRemove?: boolean;
  /** Callback when the remove button is clicked */
  onRemove?: () => void;
}

export function DataTableDateRangeFilter<TData>({
  column,
  title,
  presets,
  showRemove = false,
  onRemove,
}: DataTableDateRangeFilterProps<TData>) {
  const filterValue = column?.getFilterValue() as [Date, Date] | undefined;
  const [date, setDate] = React.useState<DateRange | undefined>(
    filterValue ? { from: filterValue[0], to: filterValue[1] } : undefined,
  );

  // Sync local state with column filter value (for reset functionality)
  const currentFilterValue = column?.getFilterValue() as [Date, Date] | undefined;
  React.useEffect(() => {
    if (currentFilterValue) {
      setDate({ from: currentFilterValue[0], to: currentFilterValue[1] });
    } else {
      setDate(undefined);
    }
  }, [currentFilterValue]);

  if (!column) {
    return null;
  }

  const handleSelect = (range: DateRange | undefined) => {
    setDate(range);
    if (range?.from && range?.to) {
      column?.setFilterValue([range.from, range.to]);
    } else {
      column?.setFilterValue(undefined);
    }
  };

  const hasValue = date?.from !== undefined;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          <CalendarIcon className="mr-2 h-4 w-4" />
          {title}
          {date?.from && (
            <>
              {' : '}
              {format(date.from, 'LLL dd, y')}
              {date.to && ` - ${format(date.to, 'LLL dd, y')}`}
            </>
          )}
          {showRemove && hasValue && onRemove && (
            <>
              <span className="mx-2 text-muted-foreground">|</span>
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
      <PopoverContent className="w-auto p-0" align="start">
        <div className="flex">
          {presets && presets.length > 0 && (
            <div className="flex flex-col gap-1 border-r p-3">
              <div className="px-2 text-xs font-medium text-muted-foreground">Presets</div>
              {presets.map((preset) => (
                <Button
                  key={preset.label}
                  variant="ghost"
                  size="sm"
                  className="h-8 justify-start px-2 font-normal"
                  onClick={() => {
                    const [from, to] = preset.getValue();
                    handleSelect({ from, to });
                  }}
                >
                  {preset.label}
                </Button>
              ))}
              {date && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 justify-start px-2 font-normal text-muted-foreground"
                  onClick={() => handleSelect(undefined)}
                >
                  Clear
                </Button>
              )}
            </div>
          )}
          <div className="p-3 w-full">
            <Calendar
              mode="range"
              defaultMonth={date?.from}
              selected={date}
              onSelect={handleSelect}
              numberOfMonths={1}
              className="sm:w-[250px]"
            />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
