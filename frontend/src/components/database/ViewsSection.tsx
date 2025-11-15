import React, { useState } from 'react';
import type { DatabaseSchemaMetadata, ViewListItem } from '@/lib/api/types';
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
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import { Database } from 'lucide-react';
import { Input } from '@/components/ui/input';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { ScrollArea } from '@/components/ui/scroll-area';

interface ViewsSectionProps {
  schemaMetadata: DatabaseSchemaMetadata | null;
  isLoading: boolean;
}

export const ViewsSection: React.FC<ViewsSectionProps> = ({ schemaMetadata, isLoading }) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedView, setSelectedView] = useState<ViewListItem | null>(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  // Filter views based on search term
  const filteredViews =
    schemaMetadata?.views?.filter((view) =>
      view.name.toLowerCase().includes(searchTerm.toLowerCase()),
    ) || [];

  const handleViewClick = (view: ViewListItem) => {
    setSelectedView(view);
    setIsDetailOpen(true);
  };

  const totalViews = schemaMetadata?.views?.length || 0;

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Database Views</CardTitle>
              <CardDescription>
                {isLoading ? (
                  'Loading views...'
                ) : schemaMetadata ? (
                  <>
                    {totalViews} view{totalViews === 1 ? '' : 's'}
                  </>
                ) : (
                  'No schema data available'
                )}
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {/* Search input */}
          {!isLoading &&
            schemaMetadata &&
            schemaMetadata.views &&
            schemaMetadata.views.length > 0 && (
              <div className="mb-4">
                <Input
                  placeholder="Search views..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="max-w-sm"
                />
              </div>
            )}

          {/* Empty state */}
          {!isLoading && filteredViews.length === 0 ? (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Database className="h-12 w-12 text-muted-foreground" />
                </EmptyMedia>
                <EmptyTitle>{searchTerm ? 'No Views Found' : 'No Views in Database'}</EmptyTitle>
                <EmptyDescription>
                  {searchTerm
                    ? `No views match the search term "${searchTerm}".`
                    : 'This database does not contain any views yet.'}
                </EmptyDescription>
              </EmptyHeader>
              {searchTerm && (
                <EmptyContent>
                  <button
                    onClick={() => setSearchTerm('')}
                    className="text-sm text-primary hover:underline"
                  >
                    Clear search
                  </button>
                </EmptyContent>
              )}
            </Empty>
          ) : (
            <div className="overflow-x-auto -mx-6 px-6">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>View Name</TableHead>
                    <TableHead>Comment</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredViews.map((view: ViewListItem) => (
                    <TableRow
                      key={view.name}
                      onClick={() => handleViewClick(view)}
                      className="cursor-pointer hover:bg-muted/50 transition-colors"
                    >
                      <TableCell className="font-medium">
                        <span className="font-mono">{view.name}</span>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground max-w-[400px] truncate">
                        {view.comment || '-'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* View Detail Sheet */}
      <Sheet open={isDetailOpen} onOpenChange={setIsDetailOpen}>
        <SheetContent side="right" className="w-full sm:max-w-3xl overflow-y-auto">
          <SheetHeader>
            <SheetTitle className="font-mono">{selectedView?.name || 'View Details'}</SheetTitle>
            <SheetDescription>View definition and metadata</SheetDescription>
          </SheetHeader>

          {selectedView && (
            <div className="mt-6 space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>View Information</CardTitle>
                  <CardDescription>General information about this view</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-1">View Name</p>
                      <p className="text-sm font-mono">{selectedView.name}</p>
                    </div>

                    {selectedView.comment && (
                      <div>
                        <p className="text-sm font-medium text-muted-foreground mb-1">Comment</p>
                        <p className="text-sm">{selectedView.comment}</p>
                      </div>
                    )}

                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-1">Definition</p>
                      <ScrollArea className="h-[400px] w-full rounded-md border">
                        <pre className="p-4 text-xs font-mono whitespace-pre-wrap break-all">
                          {selectedView.definition}
                        </pre>
                      </ScrollArea>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </SheetContent>
      </Sheet>
    </>
  );
};
