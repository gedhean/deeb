import React from 'react';
import type {
  DatabaseSchemaMetadata,
  TableListItem,
  ViewListItem,
  MaterializedViewListItem,
  ExtensionListItem,
  FunctionListItem,
  SequenceListItem,
} from '@/lib/api/types';
import { MetadataSection, DefinitionDisplay, InfoGrid } from './MetadataSection';
import { Badge } from '@/components/ui/badge';
import { Check, X } from 'lucide-react';

interface DatabaseMetadataSectionsProps {
  schemaMetadata: DatabaseSchemaMetadata | null;
  isLoading: boolean;
  onTableClick?: (tableName: string) => void;
}

export const DatabaseMetadataSections: React.FC<DatabaseMetadataSectionsProps> = ({
  schemaMetadata,
  isLoading,
  onTableClick,
}) => {
  const isPostgreSQL = schemaMetadata?.database_type?.toLowerCase() === 'postgresql';
  const isMySQL = schemaMetadata?.database_type?.toLowerCase() === 'mysql';

  // Format bytes to human-readable format
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
  };

  // Format number with commas
  const formatNumber = (num: number): string => {
    return num.toLocaleString();
  };

  return (
    <>
      {/* Tables Section */}
      {schemaMetadata?.tables && (
        <MetadataSection<TableListItem>
          title="Database Tables"
          description={`${schemaMetadata.tables.length} table(s)`}
          items={schemaMetadata.tables}
          isLoading={isLoading}
          searchKey="name"
          searchPlaceholder="Search tables..."
          emptyMessage="No Tables"
          columns={[
            {
              header: 'Table Name',
              accessor: (item) => (
                <div className="flex flex-col">
                  <span className="font-mono font-medium">{item.name}</span>
                  {item.comment && (
                    <span className="text-xs text-muted-foreground truncate max-w-[300px]">
                      {item.comment}
                    </span>
                  )}
                </div>
              ),
            },
            {
              header: 'Rows',
              accessor: (item) => (
                <span className="text-right tabular-nums">{formatNumber(item.row_count)}</span>
              ),
              width: 'text-right',
            },
            {
              header: 'Data Size',
              accessor: (item) => (
                <span className="text-right tabular-nums">{formatBytes(item.data_size)}</span>
              ),
              width: 'text-right',
            },
            {
              header: 'Index Size',
              accessor: (item) => (
                <span className="text-right tabular-nums">{formatBytes(item.index_size)}</span>
              ),
              width: 'text-right',
            },
            {
              header: 'Total Size',
              accessor: (item) => (
                <span className="text-right tabular-nums font-medium">
                  {formatBytes(item.total_size)}
                </span>
              ),
              width: 'text-right',
            },
            ...(isMySQL
              ? [
                  {
                    header: 'Engine',
                    accessor: (item: TableListItem) => (
                      <span className="text-muted-foreground">{item.engine || '-'}</span>
                    ),
                  },
                ]
              : []),
          ]}
          onItemClick={onTableClick ? (item) => onTableClick(item.name) : undefined}
        />
      )}

      {/* Views Section */}
      {schemaMetadata?.views && (
        <MetadataSection<ViewListItem>
          title="Database Views"
          description={`${schemaMetadata.views.length} view(s)`}
          items={schemaMetadata.views}
          isLoading={isLoading}
          searchKey="name"
          searchPlaceholder="Search views..."
          emptyMessage="No Views"
          columns={[
            {
              header: 'View Name',
              accessor: (item) => <span className="font-mono font-medium">{item.name}</span>,
            },
            {
              header: 'Comment',
              accessor: (item) => (
                <span className="text-sm text-muted-foreground max-w-[400px] truncate">
                  {item.comment || '-'}
                </span>
              ),
            },
          ]}
          detailRenderer={(item) => (
            <div className="space-y-4">
              <InfoGrid
                items={[
                  { label: 'Name', value: <span className="font-mono">{item.name}</span> },
                  ...(item.comment ? [{ label: 'Comment', value: item.comment }] : []),
                ]}
              />
              <DefinitionDisplay title="View Definition" content={item.definition} />
            </div>
          )}
        />
      )}

      {/* Materialized Views - PostgreSQL only */}
      {isPostgreSQL && schemaMetadata?.materialized_views && (
        <MetadataSection<MaterializedViewListItem>
          title="Materialized Views"
          description={`${schemaMetadata.materialized_views.length} materialized view(s)`}
          items={schemaMetadata.materialized_views}
          isLoading={isLoading}
          searchKey="name"
          searchPlaceholder="Search materialized views..."
          emptyMessage="No Materialized Views"
          columns={[
            {
              header: 'Name',
              accessor: (item) => <span className="font-mono font-medium">{item.name}</span>,
            },
            {
              header: 'Comment',
              accessor: (item) => (
                <span className="text-sm text-muted-foreground max-w-[400px] truncate">
                  {item.comment || '-'}
                </span>
              ),
            },
          ]}
          detailRenderer={(item) => (
            <div className="space-y-4">
              <InfoGrid
                items={[
                  { label: 'Name', value: <span className="font-mono">{item.name}</span> },
                  ...(item.comment ? [{ label: 'Comment', value: item.comment }] : []),
                ]}
              />
              <DefinitionDisplay title="View Definition" content={item.definition} />
            </div>
          )}
        />
      )}

      {/* Extensions - PostgreSQL only */}
      {isPostgreSQL && schemaMetadata?.extensions && (
        <MetadataSection<ExtensionListItem>
          title="Extensions"
          description={`${schemaMetadata.extensions.length} extension(s)`}
          items={schemaMetadata.extensions}
          isLoading={isLoading}
          searchKey="name"
          searchPlaceholder="Search extensions..."
          emptyMessage="No Extensions"
          columns={[
            {
              header: 'Name',
              accessor: (item) => <span className="font-mono font-medium">{item.name}</span>,
            },
            {
              header: 'Version',
              accessor: (item) => <Badge variant="outline">{item.version}</Badge>,
            },
            {
              header: 'Schema',
              accessor: (item) => <span className="text-sm">{item.schema}</span>,
            },
            {
              header: 'Description',
              accessor: (item) => (
                <span className="text-sm text-muted-foreground max-w-[300px] truncate">
                  {item.description || '-'}
                </span>
              ),
            },
          ]}
          detailRenderer={(item) => (
            <InfoGrid
              items={[
                { label: 'Name', value: <span className="font-mono">{item.name}</span> },
                { label: 'Version', value: <Badge variant="outline">{item.version}</Badge> },
                { label: 'Schema', value: item.schema },
                ...(item.description ? [{ label: 'Description', value: item.description }] : []),
              ]}
            />
          )}
        />
      )}

      {/* Functions */}
      {schemaMetadata?.functions && schemaMetadata.functions.length > 0 && (
        <MetadataSection<FunctionListItem>
          title="Functions"
          description={`${schemaMetadata.functions.length} function(s)`}
          items={schemaMetadata.functions}
          isLoading={isLoading}
          searchKey="name"
          searchPlaceholder="Search functions..."
          emptyMessage="No Functions"
          columns={[
            {
              header: 'Name',
              accessor: (item) => <span className="font-mono font-medium">{item.name}</span>,
            },
            {
              header: 'Comment',
              accessor: (item) => (
                <span className="text-sm text-muted-foreground max-w-[400px] truncate">
                  {item.comment || '-'}
                </span>
              ),
            },
          ]}
          detailRenderer={(item) => (
            <div className="space-y-4">
              <InfoGrid
                items={[
                  { label: 'Name', value: <span className="font-mono">{item.name}</span> },
                  ...(item.comment ? [{ label: 'Comment', value: item.comment }] : []),
                ]}
              />
              <DefinitionDisplay title="Function Definition" content={item.definition} />
            </div>
          )}
        />
      )}

      {/* Sequences - PostgreSQL only */}
      {isPostgreSQL && schemaMetadata?.sequences && (
        <MetadataSection<SequenceListItem>
          title="Sequences"
          description={`${schemaMetadata.sequences.length} sequence(s)`}
          items={schemaMetadata.sequences}
          isLoading={isLoading}
          searchKey="name"
          searchPlaceholder="Search sequences..."
          emptyMessage="No Sequences"
          columns={[
            {
              header: 'Name',
              accessor: (item) => <span className="font-mono font-medium">{item.name}</span>,
            },
            {
              header: 'Data Type',
              accessor: (item) => <Badge variant="secondary">{item.data_type}</Badge>,
            },
            {
              header: 'Increment',
              accessor: (item) => <span className="text-sm tabular-nums">{item.increment}</span>,
            },
            {
              header: 'Cycle',
              accessor: (item) =>
                item.cycle ? (
                  <Check className="h-4 w-4 text-green-500" />
                ) : (
                  <X className="h-4 w-4 text-red-500" />
                ),
            },
          ]}
          detailRenderer={(item) => (
            <div className="space-y-4">
              <InfoGrid
                items={[
                  { label: 'Name', value: <span className="font-mono">{item.name}</span> },
                  {
                    label: 'Data Type',
                    value: <Badge variant="secondary">{item.data_type}</Badge>,
                  },
                  { label: 'Start Value', value: item.start.toLocaleString() },
                  { label: 'Increment', value: item.increment.toLocaleString() },
                  { label: 'Minimum Value', value: item.min.toLocaleString() },
                  { label: 'Maximum Value', value: item.max.toLocaleString() },
                  { label: 'Cache Size', value: item.cache.toLocaleString() },
                  {
                    label: 'Cycle',
                    value: item.cycle ? (
                      <Badge variant="default">Yes</Badge>
                    ) : (
                      <Badge variant="outline">No</Badge>
                    ),
                  },
                  ...(item.comment ? [{ label: 'Comment', value: item.comment }] : []),
                ]}
              />
            </div>
          )}
        />
      )}
    </>
  );
};
