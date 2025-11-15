import React, { useEffect, useState } from 'react';
import { schemaService } from '@/lib/api/services/schema';
import type { TableDetail, Column, Index, ForeignKey, CheckConstraint } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { AlertTriangle, Check, X } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { CodeBlock } from '@/components/ui/code-block';

interface TableDetailSheetProps {
  databaseId: string;
  tableName: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Format bytes to human-readable format
 */
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
};

/**
 * Format number with thousands separator
 */
const formatNumber = (num: number): string => {
  return num.toLocaleString('en-US');
};

export const TableDetailSheet: React.FC<TableDetailSheetProps> = ({
  databaseId,
  tableName,
  open,
  onOpenChange,
}) => {
  const [tableDetail, setTableDetail] = useState<TableDetail | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (open && tableName) {
      const loadTableDetail = async () => {
        try {
          setIsLoading(true);
          setError('');
          const data = await schemaService.getTableDetail(databaseId, tableName);
          setTableDetail(data);
        } catch (err) {
          if (err instanceof APIError) {
            setError(err.message);
          } else {
            setError('Failed to load table details. Please try again.');
          }
          console.error('Failed to load table detail:', err);
        } finally {
          setIsLoading(false);
        }
      };

      loadTableDetail();
    } else {
      setTableDetail(null);
      setError('');
    }
  }, [databaseId, tableName, open]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-4xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="font-mono">{tableName || 'Table Details'}</SheetTitle>
          <SheetDescription>
            {isLoading ? 'Loading table details...' : 'Detailed information about this table'}
          </SheetDescription>
        </SheetHeader>

        {/* Error message */}
        {error && (
          <div className="mt-4 p-4 border border-destructive rounded-md">
            <div className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          </div>
        )}

        {/* Loading state */}
        {isLoading ? (
          <div className="mt-6 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>
                  <Skeleton className="h-6 w-32" />
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-3/4" />
                  <Skeleton className="h-4 w-1/2" />
                </div>
              </CardContent>
            </Card>
          </div>
        ) : (
          tableDetail && (
            <div className="mt-6">
              <Tabs defaultValue="metadata" className="w-full">
                <TabsList className="grid w-full grid-cols-4">
                  <TabsTrigger value="metadata">Metadata</TabsTrigger>
                  <TabsTrigger value="columns">Columns ({tableDetail.columns.length})</TabsTrigger>
                  <TabsTrigger value="indexes">Indexes ({tableDetail.indexes.length})</TabsTrigger>
                  <TabsTrigger value="constraints">
                    Constraints (
                    {tableDetail.foreign_keys.length + tableDetail.check_constraints.length})
                  </TabsTrigger>
                </TabsList>

                {/* Metadata Tab */}
                <TabsContent value="metadata" className="space-y-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>Table Information</CardTitle>
                      <CardDescription>General information about this table</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="grid gap-4 md:grid-cols-2">
                        <div className="space-y-1">
                          <p className="text-sm font-medium text-muted-foreground">Table Name</p>
                          <p className="text-sm font-mono">{tableDetail.name}</p>
                        </div>
                        {tableDetail.engine && (
                          <div className="space-y-1">
                            <p className="text-sm font-medium text-muted-foreground">Engine</p>
                            <p className="text-sm">{tableDetail.engine}</p>
                          </div>
                        )}
                        {tableDetail.collation && (
                          <div className="space-y-1">
                            <p className="text-sm font-medium text-muted-foreground">Collation</p>
                            <p className="text-sm font-mono">{tableDetail.collation}</p>
                          </div>
                        )}
                        {tableDetail.charset && (
                          <div className="space-y-1">
                            <p className="text-sm font-medium text-muted-foreground">Charset</p>
                            <p className="text-sm">{tableDetail.charset}</p>
                          </div>
                        )}
                        <div className="space-y-1">
                          <p className="text-sm font-medium text-muted-foreground">Row Count</p>
                          <p className="text-sm tabular-nums">
                            {formatNumber(tableDetail.row_count)}
                          </p>
                        </div>
                        <div className="space-y-1">
                          <p className="text-sm font-medium text-muted-foreground">Data Size</p>
                          <p className="text-sm tabular-nums">
                            {formatBytes(tableDetail.data_size)}
                          </p>
                        </div>
                        <div className="space-y-1">
                          <p className="text-sm font-medium text-muted-foreground">Index Size</p>
                          <p className="text-sm tabular-nums">
                            {formatBytes(tableDetail.index_size)}
                          </p>
                        </div>
                        <div className="space-y-1">
                          <p className="text-sm font-medium text-muted-foreground">Total Size</p>
                          <p className="text-sm tabular-nums font-medium">
                            {formatBytes(tableDetail.total_size)}
                          </p>
                        </div>
                        {tableDetail.comment && (
                          <div className="space-y-1 md:col-span-2">
                            <p className="text-sm font-medium text-muted-foreground">Comment</p>
                            <p className="text-sm">{tableDetail.comment}</p>
                          </div>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </TabsContent>

                {/* Columns Tab */}
                <TabsContent value="columns">
                  <Card>
                    <CardHeader>
                      <CardTitle>Columns</CardTitle>
                      <CardDescription>
                        {tableDetail.columns.length} column
                        {tableDetail.columns.length === 1 ? '' : 's'} in this table
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="overflow-x-auto">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead>Name</TableHead>
                              <TableHead>Type</TableHead>
                              <TableHead>Nullable</TableHead>
                              <TableHead>Default</TableHead>
                              <TableHead>Comment</TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {tableDetail.columns.map((column: Column) => (
                              <TableRow key={column.name}>
                                <TableCell className="font-mono font-medium">
                                  <div className="flex items-center gap-2">
                                    {column.name}
                                    {column.is_identity && (
                                      <Badge variant="secondary" className="text-xs">
                                        IDENTITY
                                      </Badge>
                                    )}
                                    {column.is_generated && (
                                      <Badge variant="outline" className="text-xs">
                                        GENERATED
                                      </Badge>
                                    )}
                                  </div>
                                </TableCell>
                                <TableCell className="font-mono text-sm">{column.type}</TableCell>
                                <TableCell>
                                  {column.nullable ? (
                                    <Check className="h-4 w-4 text-green-500" />
                                  ) : (
                                    <X className="h-4 w-4 text-red-500" />
                                  )}
                                </TableCell>
                                <TableCell className="font-mono text-sm max-w-[200px]">
                                  {column.default && column.default !== '-' ? (
                                    <Tooltip>
                                      <TooltipTrigger asChild>
                                        <span className="truncate block cursor-help">
                                          {column.default}
                                        </span>
                                      </TooltipTrigger>
                                      <TooltipContent className="max-w-md break-words">
                                        <p>{column.default}</p>
                                      </TooltipContent>
                                    </Tooltip>
                                  ) : (
                                    '-'
                                  )}
                                </TableCell>
                                <TableCell className="text-sm text-muted-foreground max-w-[200px]">
                                  {column.comment && column.comment !== '-' ? (
                                    <Tooltip>
                                      <TooltipTrigger asChild>
                                        <span className="truncate block cursor-help">
                                          {column.comment}
                                        </span>
                                      </TooltipTrigger>
                                      <TooltipContent className="max-w-md break-words">
                                        <p>{column.comment}</p>
                                      </TooltipContent>
                                    </Tooltip>
                                  ) : (
                                    '-'
                                  )}
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    </CardContent>
                  </Card>
                </TabsContent>

                {/* Indexes Tab */}
                <TabsContent value="indexes">
                  <Card>
                    <CardHeader>
                      <CardTitle>Indexes</CardTitle>
                      <CardDescription>
                        {tableDetail.indexes.length} index
                        {tableDetail.indexes.length === 1 ? '' : 'es'} on this table
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      {tableDetail.indexes.length === 0 ? (
                        <p className="text-sm text-muted-foreground">No indexes found.</p>
                      ) : (
                        <div className="space-y-4">
                          {tableDetail.indexes.map((index: Index) => (
                            <div
                              key={index.name}
                              className="p-4 border rounded-md space-y-2 hover:bg-muted/50 transition-colors"
                            >
                              <div className="flex items-center justify-between">
                                <p className="font-mono font-medium">{index.name}</p>
                                <div className="flex gap-2">
                                  {index.primary && (
                                    <Badge variant="default" className="text-xs">
                                      PRIMARY
                                    </Badge>
                                  )}
                                  {index.unique && (
                                    <Badge variant="secondary" className="text-xs">
                                      UNIQUE
                                    </Badge>
                                  )}
                                  <Badge variant="outline" className="text-xs">
                                    {index.type}
                                  </Badge>
                                </div>
                              </div>
                              <div className="text-sm text-muted-foreground">
                                <span className="font-medium">Columns: </span>
                                <span className="font-mono break-words">
                                  {index.columns.join(', ')}
                                </span>
                              </div>
                              {index.where_clause && (
                                <div className="mt-2">
                                  <div className="text-sm font-medium mb-1">Where Clause:</div>
                                  <CodeBlock language="sql" filename="" code={index.where_clause} />
                                </div>
                              )}
                              {index.comment && (
                                <div className="text-sm text-muted-foreground">{index.comment}</div>
                              )}
                            </div>
                          ))}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </TabsContent>

                {/* Constraints Tab */}
                <TabsContent value="constraints" className="space-y-4">
                  {/* Foreign Keys */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Foreign Keys</CardTitle>
                      <CardDescription>
                        {tableDetail.foreign_keys.length} foreign key
                        {tableDetail.foreign_keys.length === 1 ? '' : 's'}
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      {tableDetail.foreign_keys.length === 0 ? (
                        <p className="text-sm text-muted-foreground">No foreign keys found.</p>
                      ) : (
                        <div className="space-y-4">
                          {tableDetail.foreign_keys.map((fk: ForeignKey) => (
                            <div
                              key={fk.name}
                              className="p-4 border rounded-md space-y-2 hover:bg-muted/50 transition-colors"
                            >
                              <p className="font-mono font-medium break-words">{fk.name}</p>
                              <div className="text-sm space-y-1">
                                <div>
                                  <span className="font-medium">Columns: </span>
                                  <span className="font-mono text-muted-foreground break-words">
                                    {fk.columns.join(', ')}
                                  </span>
                                </div>
                                <div>
                                  <span className="font-medium">References: </span>
                                  <span className="font-mono text-muted-foreground break-words">
                                    {fk.referenced_table} ({fk.referenced_columns.join(', ')})
                                  </span>
                                </div>
                                <div className="flex gap-4">
                                  <div>
                                    <span className="font-medium">On Delete: </span>
                                    <Badge variant="outline" className="text-xs">
                                      {fk.on_delete}
                                    </Badge>
                                  </div>
                                  <div>
                                    <span className="font-medium">On Update: </span>
                                    <Badge variant="outline" className="text-xs">
                                      {fk.on_update}
                                    </Badge>
                                  </div>
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </CardContent>
                  </Card>

                  {/* Check Constraints */}
                  <Card>
                    <CardHeader>
                      <CardTitle>Check Constraints</CardTitle>
                      <CardDescription>
                        {tableDetail.check_constraints.length} check constraint
                        {tableDetail.check_constraints.length === 1 ? '' : 's'}
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      {tableDetail.check_constraints.length === 0 ? (
                        <p className="text-sm text-muted-foreground">No check constraints found.</p>
                      ) : (
                        <div className="space-y-4">
                          {tableDetail.check_constraints.map((cc: CheckConstraint) => (
                            <div
                              key={cc.name}
                              className="p-4 border rounded-md space-y-2 hover:bg-muted/50 transition-colors"
                            >
                              <p className="font-mono font-medium break-words">{cc.name}</p>
                              <div className="text-sm">
                                <div className="font-medium mb-1">Expression:</div>
                                <CodeBlock language="sql" filename="" code={cc.expression} />
                              </div>
                            </div>
                          ))}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </TabsContent>
              </Tabs>
            </div>
          )
        )}
      </SheetContent>
    </Sheet>
  );
};
