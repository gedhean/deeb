import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import { Database, RefreshCw } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { CodeBlock } from '@/components/ui/code-block';

interface Column<T> {
  header: string;
  accessor: (item: T) => React.ReactNode;
  width?: string;
}

interface MetadataSectionProps<T> {
  title: string;
  description?: string;
  items: T[] | undefined;
  isLoading: boolean;
  columns: Column<T>[];
  searchPlaceholder?: string;
  searchKey: keyof T;
  emptyMessage?: string;
  detailRenderer?: (item: T) => React.ReactNode;
  onItemClick?: (item: T) => void;
  onRefresh?: () => void;
  isRefreshing?: boolean;
}

export function MetadataSection<T extends { name: string }>({
  title,
  description,
  items,
  isLoading,
  columns,
  searchPlaceholder = 'Search...',
  searchKey,
  emptyMessage,
  detailRenderer,
  onItemClick,
  onRefresh,
  isRefreshing = false,
}: MetadataSectionProps<T>) {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedItem, setSelectedItem] = useState<T | null>(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  const filteredItems =
    items?.filter((item) =>
      String(item[searchKey]).toLowerCase().includes(searchTerm.toLowerCase()),
    ) || [];

  const handleItemClick = (item: T) => {
    if (onItemClick) {
      onItemClick(item);
    } else if (detailRenderer) {
      setSelectedItem(item);
      setIsDetailOpen(true);
    }
  };

  const totalItems = items?.length || 0;

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1">
              <CardTitle>{title}</CardTitle>
              {description && (
                <CardDescription>
                  {isLoading
                    ? `Loading ${title.toLowerCase()}...`
                    : `${totalItems} item${totalItems === 1 ? '' : 's'}`}
                </CardDescription>
              )}
            </div>
            <div className="flex items-center gap-2">
              {/* Search input in header */}
              {!isLoading && items && items.length > 0 && (
                <Input
                  placeholder={searchPlaceholder}
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-64"
                />
              )}
              {/* Refresh button */}
              {onRefresh && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={onRefresh}
                  disabled={isLoading || isRefreshing}
                >
                  <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {/* Empty state */}
          {!isLoading && filteredItems.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Database className="h-12 w-12 text-muted-foreground" />
                </EmptyMedia>
                <EmptyTitle>
                  {searchTerm ? `No ${title} Found` : emptyMessage || `No ${title} Available`}
                </EmptyTitle>
                <EmptyDescription>
                  {searchTerm
                    ? `No ${title.toLowerCase()} match the search term "${searchTerm}".`
                    : `This database does not contain any ${title.toLowerCase()} yet.`}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            !isLoading && (
              <div className="overflow-x-auto -mx-6 px-6">
                <Table>
                  <TableHeader>
                    <TableRow>
                      {columns.map((col, idx) => (
                        <TableHead key={idx} className={col.width}>
                          {col.header}
                        </TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredItems.map((item) => (
                      <TableRow
                        key={item.name}
                        onClick={() => handleItemClick(item)}
                        className={
                          detailRenderer || onItemClick
                            ? 'cursor-pointer hover:bg-muted/50 transition-colors'
                            : ''
                        }
                      >
                        {columns.map((col, idx) => (
                          <TableCell key={idx} className={col.width}>
                            {col.accessor(item)}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )
          )}
        </CardContent>
      </Card>

      {/* Detail Sheet */}
      {detailRenderer && selectedItem && (
        <Sheet open={isDetailOpen} onOpenChange={setIsDetailOpen}>
          <SheetContent side="right" className="w-full sm:max-w-3xl overflow-y-auto">
            <SheetHeader>
              <SheetTitle className="font-mono">{selectedItem.name}</SheetTitle>
              <SheetDescription>Detailed information</SheetDescription>
            </SheetHeader>
            <div className="mt-6">{detailRenderer(selectedItem)}</div>
          </SheetContent>
        </Sheet>
      )}
    </>
  );
}

// Helper component for displaying definition/code
export const DefinitionDisplay: React.FC<{ title: string; content: string }> = ({
  title,
  content,
}) => (
  <div className="space-y-2">
    <h3 className="text-sm font-semibold">{title}</h3>
    <CodeBlock language="sql" filename="" code={content} />
  </div>
);

// Helper component for displaying key-value pairs
export const InfoGrid: React.FC<{ items: { label: string; value: React.ReactNode }[] }> = ({
  items,
}) => (
  <Card>
    <CardHeader>
      <CardTitle>Information</CardTitle>
    </CardHeader>
    <CardContent>
      <div className="grid gap-4 md:grid-cols-2">
        {items.map((item, idx) => (
          <div key={idx} className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">{item.label}</p>
            <div className="text-sm">{item.value}</div>
          </div>
        ))}
      </div>
    </CardContent>
  </Card>
);
