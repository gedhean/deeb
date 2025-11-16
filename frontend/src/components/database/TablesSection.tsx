import React, { useState } from 'react';
import { useDatabaseSchemaSWR, useRefreshDatabaseSchema } from '@/hooks/api/useDatabaseSchemaSWR';
import type { TableListItem } from '@/lib/api/types';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
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
import { AlertTriangle, Database, RefreshCw } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

interface TablesSectionProps {
  databaseId: string;
  onTableClick: (tableName: string) => void;
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

export const TablesSection: React.FC<TablesSectionProps> = ({ databaseId, onTableClick }) => {
  // Use SWR hooks for automatic caching and revalidation
  const { schemaMetadata, isLoading, error, refetch } = useDatabaseSchemaSWR(databaseId);
  const { refreshDatabaseSchema, isRefreshing, error: refreshError } = useRefreshDatabaseSchema();
  const [searchTerm, setSearchTerm] = useState('');

  const handleRefresh = async () => {
    const result = await refreshDatabaseSchema(databaseId);
    if (result) {
      await refetch();
    }
  };

  // Combined error message
  const errorMessage = error || refreshError || '';

  // Filter tables based on search term
  const filteredTables =
    schemaMetadata?.tables.filter((table) =>
      table.name.toLowerCase().includes(searchTerm.toLowerCase()),
    ) || [];

  // Calculate totals
  const totalTables = schemaMetadata?.tables.length || 0;
  const totalRows = schemaMetadata?.tables.reduce((sum, table) => sum + table.row_count, 0) || 0;
  const totalSize = schemaMetadata?.tables.reduce((sum, table) => sum + table.total_size, 0) || 0;

  // Check if database is MySQL to show Engine column
  const isMySQLDatabase = schemaMetadata?.database_type.toLowerCase() === 'mysql';

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Database Tables</CardTitle>
            <CardDescription>
              {isLoading ? (
                'Loading tables...'
              ) : schemaMetadata ? (
                <>
                  {totalTables} table{totalTables === 1 ? '' : 's'} • {formatNumber(totalRows)}{' '}
                  total rows • {formatBytes(totalSize)} total size
                  {schemaMetadata.last_synced_at && (
                    <span className="text-xs text-muted-foreground ml-2">
                      (Last synced: {new Date(schemaMetadata.last_synced_at).toLocaleString()})
                    </span>
                  )}
                </>
              ) : (
                'No schema data available'
              )}
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={isLoading || isRefreshing}
            className="ml-4"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? 'animate-spin' : ''}`} />
            {isRefreshing ? 'Refreshing...' : 'Refresh'}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {/* Error message */}
        {errorMessage && (
          <div className="mb-4 p-4 border border-destructive rounded-md">
            <div className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-4 w-4" />
              <span className="text-sm">{errorMessage}</span>
            </div>
          </div>
        )}

        {/* Search input */}
        {!isLoading && schemaMetadata && schemaMetadata.tables.length > 0 && (
          <div className="mb-4">
            <Input
              placeholder="Search tables..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="max-w-sm"
            />
          </div>
        )}

        {/* Loading state */}
        {isLoading ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Table Name</TableHead>
                <TableHead className="text-right">Rows</TableHead>
                <TableHead className="text-right">Data Size</TableHead>
                <TableHead className="text-right">Index Size</TableHead>
                <TableHead className="text-right">Total Size</TableHead>
                {isMySQLDatabase && <TableHead>Engine</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {[1, 2, 3, 4, 5].map((i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Skeleton className="h-4 w-48" />
                  </TableCell>
                  <TableCell className="text-right">
                    <Skeleton className="h-4 w-16 ml-auto" />
                  </TableCell>
                  <TableCell className="text-right">
                    <Skeleton className="h-4 w-20 ml-auto" />
                  </TableCell>
                  <TableCell className="text-right">
                    <Skeleton className="h-4 w-20 ml-auto" />
                  </TableCell>
                  <TableCell className="text-right">
                    <Skeleton className="h-4 w-20 ml-auto" />
                  </TableCell>
                  {isMySQLDatabase && (
                    <TableCell>
                      <Skeleton className="h-4 w-16" />
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : filteredTables.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Database className="h-12 w-12 text-muted-foreground" />
              </EmptyMedia>
              <EmptyTitle>{searchTerm ? 'No Tables Found' : 'No Tables in Database'}</EmptyTitle>
              <EmptyDescription>
                {searchTerm
                  ? `No tables match the search term "${searchTerm}".`
                  : 'This database does not contain any tables yet.'}
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
                  <TableHead>Table Name</TableHead>
                  <TableHead className="text-right">Rows</TableHead>
                  <TableHead className="text-right">Data Size</TableHead>
                  <TableHead className="text-right">Index Size</TableHead>
                  <TableHead className="text-right">Total Size</TableHead>
                  {isMySQLDatabase && <TableHead>Engine</TableHead>}
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredTables.map((table: TableListItem) => (
                  <TableRow
                    key={table.name}
                    onClick={() => onTableClick(table.name)}
                    className="cursor-pointer hover:bg-muted/50 transition-colors"
                  >
                    <TableCell className="font-medium">
                      <div className="flex flex-col">
                        <span className="font-mono">{table.name}</span>
                        {table.comment && (
                          <span className="text-xs text-muted-foreground truncate max-w-[300px]">
                            {table.comment}
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {formatNumber(table.row_count)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {formatBytes(table.data_size)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {formatBytes(table.index_size)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums font-medium">
                      {formatBytes(table.total_size)}
                    </TableCell>
                    {isMySQLDatabase && (
                      <TableCell className="text-muted-foreground">{table.engine || '-'}</TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
