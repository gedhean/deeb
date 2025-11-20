/**
 * DataTable Component
 *
 * A reusable data table component built with TanStack Table that provides:
 * - Pagination (client-side and server-side)
 * - Sorting
 * - Filtering
 * - Column visibility
 * - Row selection (optional)
 * - Loading and empty states
 *
 * @example
 * ```tsx
 * const columns: ColumnDef<User>[] = [
 *   {
 *     accessorKey: "email",
 *     header: "Email",
 *   },
 *   {
 *     accessorKey: "name",
 *     header: "Name",
 *   },
 * ];
 *
 * <DataTable
 *   columns={columns}
 *   data={users}
 *   pagination={{
 *     pageIndex: 0,
 *     pageSize: 10,
 *     totalCount: 100,
 *     onPageChange: (pageIndex) => {},
 *   }}
 * />
 * ```
 */

import * as React from 'react';
import type {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
} from '@tanstack/react-table';
import {
  flexRender,
  getCoreRowModel,
  getFacetedMinMaxValues,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type Table as TableType,
} from '@tanstack/react-table';
import { ChevronDown } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { DataTablePagination } from './data-table-pagination';
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';

export interface DataTablePaginationOptions {
  /** Current page index (0-based) */
  pageIndex: number;
  /** Number of items per page */
  pageSize: number;
  /** Total number of items (for server-side pagination) */
  totalCount?: number;
  /** Callback when page changes */
  onPageChange?: (pageIndex: number) => void;
  /** Callback when page size changes */
  onPageSizeChange?: (pageSize: number) => void;
  /** Available page size options */
  pageSizeOptions?: number[];
  /** Enable manual filtering (disable client-side filtering for server-side filtering) */
  manualFiltering?: boolean;
}

export interface DataTableProps<TData, TValue> {
  /** Column definitions */
  columns: ColumnDef<TData, TValue>[];
  /** Table data */
  data: TData[];
  /** Whether data is loading */
  isLoading?: boolean;
  /** Initial column filters state (applied on mount) */
  initialColumnFilters?: ColumnFiltersState;
  /** Pagination options */
  pagination?: DataTablePaginationOptions;
  /**
   * Custom toolbar component passed as children (render prop)
   * Receives the table instance as a function parameter
   */
  children?: (table: TableType<TData>) => React.ReactNode;
  /**
   * @deprecated Use children instead of toolbar prop
   * Custom toolbar component
   * Receives the table instance for composition
   */
  toolbar?: (table: TableType<TData>) => React.ReactNode;
  /**
   * @deprecated Use children with composition pattern instead
   * Whether to enable search/filter
   */
  searchable?: boolean;
  /**
   * @deprecated Use children with composition pattern instead
   * Placeholder text for search input
   */
  searchPlaceholder?: string;
  /**
   * @deprecated Use children with composition pattern instead
   * Key to search/filter on
   */
  searchKey?: string;
  /**
   * @deprecated Use children with composition pattern instead
   * Whether to show column visibility toggle
   */
  showColumnVisibility?: boolean;
  /** Custom empty state message */
  emptyMessage?: string;
  /** Whether to enable row selection */
  enableRowSelection?: boolean;
  /** Callback when row selection changes */
  onRowSelectionChange?: (selectedRows: TData[]) => void;
  /** Callback when a row is clicked */
  onRowClick?: (row: TData) => void;
  /** Callback when column filters change (for server-side filtering) */
  onFiltersChange?: (filters: ColumnFiltersState) => void;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  isLoading = false,
  initialColumnFilters,
  pagination,
  children,
  toolbar,
  searchable = false,
  searchPlaceholder = 'Search...',
  searchKey,
  showColumnVisibility = true,
  emptyMessage = 'No results.',
  enableRowSelection = false,
  onRowSelectionChange,
  onRowClick,
  onFiltersChange,
}: DataTableProps<TData, TValue>) {
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
    initialColumnFilters || [],
  );
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({});
  const [rowSelection, setRowSelection] = React.useState({});

  // Calculate pagination state
  const pageCount = pagination?.totalCount
    ? Math.ceil(pagination.totalCount / pagination.pageSize)
    : undefined;

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: pagination ? undefined : getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: pagination?.manualFiltering ? undefined : getFilteredRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    getFacetedMinMaxValues: getFacetedMinMaxValues(),
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: enableRowSelection ? setRowSelection : undefined,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      ...(enableRowSelection && { rowSelection }),
      ...(pagination && {
        pagination: {
          pageIndex: pagination.pageIndex,
          pageSize: pagination.pageSize,
        },
      }),
    },
    ...(pagination && {
      manualPagination: true,
      pageCount,
    }),
    ...(pagination?.manualFiltering && {
      manualFiltering: true,
    }),
    ...(enableRowSelection && {
      enableRowSelection: true,
    }),
  });

  // Notify parent of row selection changes
  React.useEffect(() => {
    if (enableRowSelection && onRowSelectionChange) {
      const selectedRows = table.getFilteredSelectedRowModel().rows.map((row) => row.original);
      onRowSelectionChange(selectedRows);
    }
  }, [rowSelection, enableRowSelection, onRowSelectionChange, table]);

  // Notify parent of filter changes
  React.useEffect(() => {
    if (onFiltersChange) {
      onFiltersChange(columnFilters);
    }
  }, [columnFilters, onFiltersChange]);

  // Deprecation warnings for old toolbar props
  React.useEffect(() => {
    if (toolbar) {
      console.warn(
        'DataTable: toolbar prop is deprecated. ' +
          'Pass toolbar components as children instead. ' +
          'See https://www.diceui.com/docs/components/data-table.mdx for examples.',
      );
    }
    if (searchable || searchKey || searchPlaceholder !== 'Search...' || !showColumnVisibility) {
      console.warn(
        'DataTable: searchable, searchKey, searchPlaceholder, and showColumnVisibility props are deprecated. ' +
          'Pass toolbar components as children instead. ' +
          'See https://www.diceui.com/docs/components/data-table.mdx for examples.',
      );
    }
  }, [toolbar, searchable, searchKey, searchPlaceholder, showColumnVisibility]);

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      {children
        ? // New children-based toolbar (render prop)
          children(table)
        : toolbar
          ? // Deprecated: toolbar prop
            toolbar(table)
          : // Legacy toolbar (deprecated)
            (searchable || showColumnVisibility) && (
              <div className="flex items-center justify-between">
                <div className="flex flex-1 items-center space-x-2">
                  {searchable && searchKey && (
                    <Input
                      placeholder={searchPlaceholder}
                      value={(table.getColumn(searchKey)?.getFilterValue() as string) ?? ''}
                      onChange={(event) =>
                        table.getColumn(searchKey)?.setFilterValue(event.target.value)
                      }
                      className="h-8 w-[150px] lg:w-[350px]"
                    />
                  )}
                </div>
                {showColumnVisibility && (
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        className="ml-auto h-8 text-muted-foreground"
                      >
                        Columns <ChevronDown className="ml-2 h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      {table
                        .getAllColumns()
                        .filter((column) => column.getCanHide())
                        .map((column) => {
                          return (
                            <DropdownMenuCheckboxItem
                              key={column.id}
                              className="capitalize"
                              checked={column.getIsVisible()}
                              onCheckedChange={(value) => column.toggleVisibility(!!value)}
                            >
                              {column.id}
                            </DropdownMenuCheckboxItem>
                          );
                        })}
                    </DropdownMenuContent>
                  </DropdownMenu>
                )}
              </div>
            )}

      {/* Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {isLoading ? (
              // Loading state
              Array.from({ length: pagination?.pageSize ?? 10 }).map((_, index) => (
                <TableRow key={index}>
                  {columns.map((_, cellIndex) => (
                    <TableCell key={cellIndex}>
                      <Skeleton className="h-6 w-full" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : table.getRowModel().rows?.length ? (
              // Data rows
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && 'selected'}
                  onClick={() => onRowClick?.(row.original)}
                  className={onRowClick ? 'cursor-pointer hover:bg-muted/50 transition-colors' : ''}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              // Empty state
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center">
                  {emptyMessage}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {pagination && (
        <DataTablePagination
          table={table}
          pageSizeOptions={pagination.pageSizeOptions}
          totalCount={pagination.totalCount}
          pageIndex={pagination.pageIndex}
          onPageChange={pagination.onPageChange}
          onPageSizeChange={pagination.onPageSizeChange}
        />
      )}
    </div>
  );
}
