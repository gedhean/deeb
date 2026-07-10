import * as React from 'react';
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react';
import type { Table as TanStackTable } from '@tanstack/react-table';

import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface DataTablePaginationProps<TData> {
  table: TanStackTable<TData>;
  pageSizeOptions?: number[];
  totalCount?: number;
  pageIndex?: number;
  onPageChange?: (pageIndex: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
}

export function DataTablePagination<TData>({
  table,
  pageSizeOptions = [10, 20, 50, 100],
  totalCount,
  pageIndex,
  onPageChange,
  onPageSizeChange,
}: DataTablePaginationProps<TData>) {
  // Get current page size from table state
  const currentPageSize = table.getState().pagination.pageSize;

  // Calculate page count for server-side pagination
  const pageCount = React.useMemo(() => {
    if (totalCount !== undefined && pageIndex !== undefined) {
      return Math.ceil(totalCount / currentPageSize);
    }
    return table.getPageCount();
  }, [totalCount, pageIndex, currentPageSize, table]);

  const currentPage =
    pageIndex !== undefined ? pageIndex + 1 : table.getState().pagination.pageIndex + 1;
  const isFirstPage = pageIndex !== undefined ? pageIndex === 0 : !table.getCanPreviousPage();
  const isLastPage = pageIndex !== undefined ? pageIndex >= pageCount - 1 : !table.getCanNextPage();

  return (
    <div className="flex items-center justify-between px-2">
      <div className="flex-1 text-sm text-muted-foreground">
        {table.getFilteredSelectedRowModel().rows.length > 0 && (
          <span>
            {table.getFilteredSelectedRowModel().rows.length} of{' '}
            {table.getFilteredRowModel().rows.length} row(s) selected.
          </span>
        )}
      </div>
      <div className="flex items-center space-x-6 lg:space-x-8">
        <div className="flex items-center space-x-2">
          <p className="text-sm font-medium text-muted-foreground">Rows per page</p>
          <Select
            value={`${table.getState().pagination.pageSize}`}
            onValueChange={(value) => {
              const newSize = Number(value);
              if (onPageSizeChange) {
                onPageSizeChange(newSize);
              } else {
                table.setPageSize(newSize);
              }
            }}
          >
            <SelectTrigger className="h-8 w-[70px]">
              <SelectValue placeholder={table.getState().pagination.pageSize} />
            </SelectTrigger>
            <SelectContent side="top">
              {pageSizeOptions.map((pageSize) => (
                <SelectItem key={pageSize} value={`${pageSize}`}>
                  {pageSize}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex w-[100px] items-center justify-center text-sm font-medium text-muted-foreground">
          Page {currentPage} of {pageCount}
        </div>
        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            className="hidden h-8 w-8 p-0 lg:flex"
            onClick={() => {
              if (onPageChange) {
                onPageChange(0);
              } else {
                table.setPageIndex(0);
              }
            }}
            disabled={isFirstPage}
          >
            <span className="sr-only">Go to first page</span>
            <ChevronsLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="h-8 w-8 p-0"
            onClick={() => {
              if (onPageChange) {
                onPageChange(Math.max(0, (pageIndex ?? table.getState().pagination.pageIndex) - 1));
              } else {
                table.previousPage();
              }
            }}
            disabled={isFirstPage}
          >
            <span className="sr-only">Go to previous page</span>
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="h-8 w-8 p-0"
            onClick={() => {
              if (onPageChange) {
                onPageChange((pageIndex ?? table.getState().pagination.pageIndex) + 1);
              } else {
                table.nextPage();
              }
            }}
            disabled={isLastPage}
          >
            <span className="sr-only">Go to next page</span>
            <ChevronRight className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="hidden h-8 w-8 p-0 lg:flex"
            onClick={() => {
              if (onPageChange) {
                onPageChange(pageCount - 1);
              } else {
                table.setPageIndex(pageCount - 1);
              }
            }}
            disabled={isLastPage}
          >
            <span className="sr-only">Go to last page</span>
            <ChevronsRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
