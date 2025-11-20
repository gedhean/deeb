import React, { useEffect, useRef, useState, useMemo } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useUsers } from '@/hooks/api/useUsersSWR';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useDatabaseSWR } from '@/hooks/api/useDatabaseSWR';
import { useEnvironmentsSWR } from '@/hooks/api/useEnvironmentsSWR';
import { useDatabaseInstancesSWR } from '@/hooks/api/useDatabaseInstancesSWR';
import { useChangePlansSWR } from '@/hooks/api/useChangePlansSWR';
import { useDatabaseSchemaSWR, useRefreshDatabaseSchema } from '@/hooks/api/useDatabaseSchemaSWR';
import type { Project, Environment, DatabaseInstance } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { RefreshCw } from 'lucide-react';
import { toast } from 'sonner';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ArrowLeft, AlertTriangle, ExternalLink, FileEdit } from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';
import { ChangePlanStatusBadge } from '@/components/ChangePlanStatusBadge';
import { DatabaseInfoCard } from '@/components/database/DatabaseInfoCard';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';
import { TableDetailSheet } from '@/components/database/TableDetailSheet';
import { DatabaseMetadataSections } from '@/components/database/DatabaseMetadataSections';
import { DataTable } from '@/components/data-table/data-table';
import { changelogColumns } from './databases/changelog-columns';
import { changelogService } from '@/lib/api/services/changelog';
import useSWR from 'swr';

export const DatabaseDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const [selectedTableName, setSelectedTableName] = useState<string | null>(null);
  const [isTableDetailOpen, setIsTableDetailOpen] = useState(false);
  const [isRefreshDisabled, setIsRefreshDisabled] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Pagination state for changelog table
  const [changelogPageIndex, setChangelogPageIndex] = useState(0);
  const [changelogPageSize, setChangelogPageSize] = useState(10);

  // Use cached data from SWR hooks
  const { getUserById } = useUsers();
  const { getProjectById } = useProjectsSWR();
  const { environments } = useEnvironmentsSWR();
  const { databaseInstances } = useDatabaseInstancesSWR();

  // Fetch database with SWR
  const { database, isLoading, error: dbError } = useDatabaseSWR(id);

  // Fetch schema metadata with SWR
  const {
    schemaMetadata,
    isLoading: isLoadingSchema,
    refetch: refetchSchema,
  } = useDatabaseSchemaSWR(database?.id);

  // Use refresh schema hook
  const { refreshDatabaseSchema, isRefreshing: isRefreshingSchema } = useRefreshDatabaseSchema();

  // Fetch change plans for this database with SWR
  const changePlansParams = useMemo(
    () => (database?.id ? { database_id: database.id, limit: 100 } : undefined),
    [database],
  );
  const { changePlans, isLoading: isLoadingChangePlans } = useChangePlansSWR(
    changePlansParams || {},
  );

  // Fetch changelog for this database with SWR
  const {
    data: changelogs,
    error: changelogsError,
    isLoading: isLoadingChangelogs,
  } = useSWR(database?.id ? `/api/v1/databases/${database.id}/changelog` : null, () =>
    database?.id ? changelogService.listDatabaseChangelog(database.id) : null,
  );

  // Compute related entities from cached data
  const project = useMemo<Project | null>(
    () => (database?.project_id ? getProjectById(database.project_id) || null : null),
    [database, getProjectById],
  );

  const environment = useMemo<Environment | null>(
    () =>
      database?.environment_id
        ? environments.find((e) => e.id === database.environment_id) || null
        : null,
    [database, environments],
  );

  const instance = useMemo<DatabaseInstance | null>(
    () =>
      database
        ? databaseInstances.find((i) => i.id === database.database_instance_id) || null
        : null,
    [database, databaseInstances],
  );

  const error = dbError || '';

  // Check if user has databases.read permission
  const canReadDatabases = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabasesRead,
  );

  // Format date
  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return dateString;
    }
  };

  // Get author name
  const getAuthorName = (authorId: string): string => {
    const author = getUserById(authorId);
    return author?.email || authorId;
  };

  // Handle table click
  const handleTableClick = (tableName: string) => {
    setSelectedTableName(tableName);
    setIsTableDetailOpen(true);
  };

  // Refresh schema metadata with debounce and toast notifications
  const handleRefreshSchema = async () => {
    if (!database?.id || isRefreshDisabled || isRefreshingSchema) return;

    // Clear any existing timer before starting a new one
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current);
      refreshTimerRef.current = null;
    }

    setIsRefreshDisabled(true);

    toast.info('Schema refresh started', {
      description: 'The refresh is running in the background and may take a few moments.',
    });

    const result = await refreshDatabaseSchema(database.id);

    if (result) {
      // Invalidate the schema cache to trigger a refetch
      await refetchSchema();
    }

    // Enable debounce - disable button for 5 seconds
    refreshTimerRef.current = setTimeout(() => {
      setIsRefreshDisabled(false);
      refreshTimerRef.current = null;
    }, 5000);
  };

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current);
        refreshTimerRef.current = null;
      }
    };
  }, []);

  // Render access denied
  if (!canReadDatabases) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Databases', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires the databases.read permission.
            </p>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout
      breadcrumbs={[
        { label: 'Databases', href: '/databases' },
        { label: database?.database_name || 'Database', isCurrentPage: true },
      ]}
    >
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => navigate('/databases')}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold tracking-tight">
                {isLoading ? (
                  <Skeleton className="h-9 w-64" />
                ) : (
                  database?.database_name || 'Database'
                )}
              </h1>
              <div className="text-muted-foreground mt-1">
                {isLoading ? (
                  <Skeleton className="h-4 w-96 mt-2" />
                ) : (
                  database?.description || 'Database details'
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Error message */}
        {error && (
          <Card className="border-destructive">
            <CardContent className="pt-6">
              <div className="flex items-center gap-2 text-destructive">
                <AlertTriangle className="h-4 w-4" />
                <span className="text-sm">{error}</span>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Tabs */}
        <Tabs
          defaultValue="overview"
          value={activeTab}
          onValueChange={setActiveTab}
          className="space-y-6"
        >
          <div className="flex items-center justify-between">
            <TabsList>
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="change-plans">Change Plans</TabsTrigger>
              <TabsTrigger value="changelog">Changelog</TabsTrigger>
            </TabsList>
            {activeTab === 'overview' && (
              <Button
                variant="outline"
                onClick={handleRefreshSchema}
                disabled={isLoadingSchema || isRefreshingSchema || isRefreshDisabled}
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshingSchema ? 'animate-spin' : ''}`} />
                {isRefreshingSchema ? 'Refreshing...' : 'Refresh Schema'}
              </Button>
            )}
            {activeTab === 'change-plans' && (
              <Button
                onClick={() => {
                  if (database?.id) {
                    navigate(`/change-plans/create?step=1&database_id=${database.id}`);
                  }
                }}
                disabled={!database?.id}
              >
                <FileEdit className="h-4 w-4 mr-2" />
                Create Change Plan
              </Button>
            )}
          </div>

          {/* Overview Tab */}
          <TabsContent value="overview" className="space-y-6">
            {/* Database Info */}
            <DatabaseInfoCard
              database={database}
              project={project}
              environment={environment}
              instance={instance}
              isLoading={isLoading}
              onNavigate={navigate}
              formatDate={formatDate}
            />

            {/* Database Metadata Sections (Tables, Views, Materialized Views, Extensions, Functions, Sequences) */}
            {!isLoading && database && (
              <>
                <DatabaseMetadataSections
                  schemaMetadata={schemaMetadata}
                  isLoading={isLoadingSchema}
                  onTableClick={handleTableClick}
                />
                <TableDetailSheet
                  databaseId={database.id}
                  tableName={selectedTableName}
                  open={isTableDetailOpen}
                  onOpenChange={setIsTableDetailOpen}
                />
              </>
            )}
          </TabsContent>

          {/* Change Plans Tab */}
          <TabsContent value="change-plans" className="space-y-6">
            {/* Database Info */}
            <DatabaseInfoCard
              database={database}
              project={project}
              environment={environment}
              instance={instance}
              isLoading={isLoading}
              onNavigate={navigate}
              formatDate={formatDate}
            />

            {/* Change Plans Table */}
            <Card>
              <CardHeader>
                <CardTitle>Change Plans</CardTitle>
                <CardDescription>
                  {isLoadingChangePlans ? (
                    'Loading change plans...'
                  ) : (
                    <>
                      {changePlans.length} change plan{changePlans.length === 1 ? '' : 's'} for this
                      database
                    </>
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {isLoadingChangePlans ? (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Title</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Change Type</TableHead>
                        <TableHead>Author</TableHead>
                        <TableHead>Created</TableHead>
                        <TableHead className="text-right">Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {[1, 2, 3].map((i) => (
                        <TableRow key={i}>
                          <TableCell>
                            <Skeleton className="h-4 w-32" />
                          </TableCell>
                          <TableCell>
                            <Skeleton className="h-4 w-24" />
                          </TableCell>
                          <TableCell>
                            <Skeleton className="h-4 w-24" />
                          </TableCell>
                          <TableCell>
                            <Skeleton className="h-4 w-32" />
                          </TableCell>
                          <TableCell>
                            <Skeleton className="h-4 w-24" />
                          </TableCell>
                          <TableCell className="text-right">
                            <Skeleton className="h-8 w-16 ml-auto" />
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                ) : changePlans.length === 0 ? (
                  <Empty>
                    <EmptyHeader>
                      <EmptyMedia variant="icon">
                        <FileEdit className="h-12 w-12 text-muted-foreground" />
                      </EmptyMedia>
                      <EmptyTitle>No Change Plans Found</EmptyTitle>
                      <EmptyDescription>
                        No change plans have been created for this database yet.
                      </EmptyDescription>
                    </EmptyHeader>
                    <EmptyContent>
                      <Button
                        onClick={() => {
                          if (database?.id) {
                            navigate(`/change-plans/create?database_id=${database.id}`);
                          }
                        }}
                        disabled={!database?.id}
                      >
                        <FileEdit className="mr-2 h-4 w-4" />
                        Create Change Plan
                      </Button>
                    </EmptyContent>
                  </Empty>
                ) : (
                  <div className="overflow-x-auto -mx-6 px-6">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Title</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead>Change Type</TableHead>
                          <TableHead>Author</TableHead>
                          <TableHead>Created</TableHead>
                          <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {changePlans.map((changePlan) => (
                          <TableRow
                            key={changePlan.id}
                            onClick={() => navigate(`/change-plans/${changePlan.id}`)}
                            className="cursor-pointer hover:bg-muted/50 transition-colors"
                          >
                            <TableCell className="font-medium">
                              <div className="flex flex-col">
                                <span className="truncate max-w-[200px]">{changePlan.title}</span>
                                {changePlan.description && (
                                  <span className="text-xs text-muted-foreground truncate max-w-[200px]">
                                    {changePlan.description}
                                  </span>
                                )}
                              </div>
                            </TableCell>
                            <TableCell>
                              <ChangePlanStatusBadge status={changePlan.status} />
                            </TableCell>
                            <TableCell>
                              <Badge variant="outline">{changePlan.change_type}</Badge>
                            </TableCell>
                            <TableCell className="text-muted-foreground">
                              {getAuthorName(changePlan.author_id)}
                            </TableCell>
                            <TableCell className="text-muted-foreground">
                              {formatDate(changePlan.created_at)}
                            </TableCell>
                            <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  navigate(`/change-plans/${changePlan.id}`);
                                }}
                                className="h-8"
                              >
                                <ExternalLink className="h-4 w-4" />
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="changelog" className="space-y-6">
            {/* Database Info */}
            <DatabaseInfoCard
              database={database}
              project={project}
              environment={environment}
              instance={instance}
              isLoading={isLoading}
              onNavigate={navigate}
              formatDate={formatDate}
            />

            {/* Changelog Table */}
            <Card>
              <CardHeader>
                <CardTitle>Database Changelog</CardTitle>
                <CardDescription>
                  {isLoadingChangelogs
                    ? 'Loading changelog...'
                    : `${changelogs?.length || 0} changelog ${(changelogs?.length || 0) === 1 ? 'entry' : 'entries'} for this database`}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {changelogsError ? (
                  <div className="flex flex-col items-center justify-center py-8 text-center">
                    <AlertTriangle className="h-12 w-12 text-destructive mb-4" />
                    <h3 className="text-lg font-semibold mb-2">Failed to Load Changelog</h3>
                    <p className="text-sm text-muted-foreground max-w-md">
                      {changelogsError instanceof Error
                        ? changelogsError.message
                        : 'An error occurred while fetching the changelog. Please try again later.'}
                    </p>
                  </div>
                ) : isLoadingChangelogs ? (
                  <div className="flex items-center justify-center py-8">
                    <Skeleton className="h-32 w-full" />
                  </div>
                ) : !changelogs || changelogs.length === 0 ? (
                  <Empty>
                    <EmptyHeader>
                      <EmptyMedia variant="icon">
                        <FileEdit className="h-12 w-12 text-muted-foreground" />
                      </EmptyMedia>
                      <EmptyTitle>No Changelog Entries</EmptyTitle>
                      <EmptyDescription>
                        No change plans have been executed for this database yet.
                      </EmptyDescription>
                    </EmptyHeader>
                  </Empty>
                ) : (
                  <DataTable
                    columns={changelogColumns}
                    data={changelogs}
                    onRowClick={(row) => navigate(`/changelog/${row.id}`)}
                    pagination={{
                      pageIndex: changelogPageIndex,
                      pageSize: changelogPageSize,
                      totalCount: changelogs.length,
                      onPageChange: setChangelogPageIndex,
                      onPageSizeChange: (newSize) => {
                        setChangelogPageSize(newSize);
                        setChangelogPageIndex(0);
                      },
                      pageSizeOptions: [10, 20, 30, 50],
                    }}
                  />
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </DefaultLayout>
  );
};
