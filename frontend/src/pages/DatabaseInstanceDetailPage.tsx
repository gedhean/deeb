import React, { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useDatabaseInstanceById } from '@/hooks/api/useDatabaseInstancesSWR';
import { useDatabases } from '@/hooks/api/useDatabases';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useEnvironmentsSWR } from '@/hooks/api/useEnvironmentsSWR';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import { DataTable } from '@/components/data-table';
import { getDatabasesColumns } from './database-instances/databases-columns';
import {
  ArrowLeft,
  AlertTriangle,
  Calendar,
  Database as DatabaseIcon,
  Plus,
  Cable,
  RefreshCw,
  CheckCircle2,
} from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';
import { CreateDatabaseDialog } from '@/components/CreateDatabaseDialog';
import { DatabaseStatusBadge } from '@/components/DatabaseStatusBadge';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import { toast } from 'sonner';

export const DatabaseInstanceDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [syncResult, setSyncResult] = useState<{
    total_found: number;
    synced: number;
    failed: number;
    skipped: number;
    failed_databases?: string[];
    skipped_databases?: string[];
  } | null>(null);

  // Pagination state for databases table
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(10);

  // Check permissions
  const canReadDatabaseInstances = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabaseInstancesRead,
  );
  const canCreateDatabases = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabasesCreate,
  );

  // Use SWR hooks for data fetching
  const {
    databaseInstance: instance,
    isLoading: isLoadingInstance,
    error: instanceError,
  } = useDatabaseInstanceById(id);

  // Fetch databases filtered by instance ID
  const databasesParams = useMemo(
    () => ({
      database_instance_id: id,
      limit: 100,
    }),
    [id],
  );
  const { databases, isLoading: isLoadingDatabases } = useDatabases(databasesParams);

  // Fetch projects and environments for display
  const { projects } = useProjectsSWR();
  const { environments } = useEnvironmentsSWR();

  // Compute combined loading and error states
  const isLoading = isLoadingInstance;
  const error = !canReadDatabaseInstances
    ? 'Access denied. You do not have permission to view database instances.'
    : instanceError || '';

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

  // Get project name
  const getProjectName = (projectId: string | null | undefined): string => {
    if (!projectId) return 'Unassigned';
    const project = projects.find((p) => p.id === projectId);
    return project?.name || 'Unknown';
  };

  // Get environment name
  const getEnvironmentName = (environmentId: string | null | undefined): string => {
    if (!environmentId) return 'Unassigned';
    const env = environments.find((e) => e.id === environmentId);
    return env?.name || 'Unknown';
  };

  // Get columns for databases table
  const databasesColumns = getDatabasesColumns({
    getProjectName,
    getEnvironmentName,
  });

  // Handle test connection
  const handleTestConnection = async () => {
    if (!id) return;

    setIsTestingConnection(true);
    try {
      const result = await databaseInstancesService.testConnection(id);

      if (result.success) {
        toast.success('Connection Successful', {
          description: `Connected to ${result.host}:${result.port}/${result.database}`,
        });
      } else {
        toast.error('Connection Failed', {
          description: result.error || 'Failed to connect to database',
        });
      }
    } catch (error) {
      toast.error('Error', {
        description: error instanceof Error ? error.message : 'Unknown error',
      });
    } finally {
      setIsTestingConnection(false);
    }
  };

  // Handle sync databases
  const handleSyncDatabases = async () => {
    if (!id) return;

    setIsSyncing(true);
    try {
      const result = await databaseInstancesService.syncDatabases(id);
      setSyncResult(result);

      // Show toast notification
      if (result.synced > 0) {
        toast.success('Sync Completed', {
          description: `Synced ${result.synced} database(s). ${result.skipped} skipped, ${result.failed} failed.`,
        });
      } else if (result.failed > 0) {
        toast.error('Sync Failed', {
          description: `Failed to sync ${result.failed} database(s). ${result.skipped} already existed.`,
        });
      } else {
        toast.info('No New Databases', {
          description: `All ${result.total_found} database(s) already exist in Deeb.`,
        });
      }
    } catch (error) {
      toast.error('Sync Error', {
        description: error instanceof Error ? error.message : 'Unknown error occurred',
      });
    } finally {
      setIsSyncing(false);
    }
  };

  // Render access denied
  if (!canReadDatabaseInstances) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Database Instances', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires the database_instances.read permission.
            </p>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout
      breadcrumbs={[
        { label: 'Database Instances', href: '/database-instances' },
        { label: instance?.name || 'Instance', isCurrentPage: true },
      ]}
    >
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => navigate('/database-instances')}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold tracking-tight">
                {isLoading ? (
                  <Skeleton className="h-9 w-64" />
                ) : (
                  instance?.name || 'Database Instance'
                )}
              </h1>
              <p className="text-muted-foreground mt-1">
                {isLoading ? (
                  <Skeleton className="h-4 w-96 mt-2" />
                ) : (
                  instance?.description || 'Database instance details'
                )}
              </p>
            </div>
          </div>
          {!isLoading && instance && (
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleTestConnection}
                disabled={isTestingConnection}
              >
                <Cable className="h-4 w-4 mr-2" />
                {isTestingConnection ? 'Testing...' : 'Test Connection'}
              </Button>
              {canCreateDatabases && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleSyncDatabases}
                  disabled={isSyncing}
                >
                  <RefreshCw className={`h-4 w-4 mr-2 ${isSyncing ? 'animate-spin' : ''}`} />
                  {isSyncing ? 'Syncing...' : 'Sync Databases'}
                </Button>
              )}
            </div>
          )}
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

        {/* Instance Info */}
        {isLoading ? (
          <Card>
            <CardHeader>
              <CardTitle>Instance Information</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            </CardContent>
          </Card>
        ) : instance ? (
          <Card>
            <CardHeader>
              <CardTitle>Instance Information</CardTitle>
              <CardDescription>Details about this database instance connection</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-3">
                {/* Column 1 */}
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Name</p>
                  <p className="text-sm">{instance.name}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Host</p>
                  <p className="text-sm font-mono">
                    {instance.host}:{instance.port}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Status</p>
                  <DatabaseStatusBadge status={instance.status} variant="instance" />
                </div>

                {/* Column 2 */}
                {instance.description && (
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-muted-foreground">Description</p>
                    <p className="text-sm">{instance.description}</p>
                  </div>
                )}
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Username</p>
                  <p className="text-sm font-mono">{instance.username}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">SSL Required</p>
                  <p className="text-sm">{instance.require_ssl ? 'Yes' : 'No'}</p>
                </div>

                {/* Column 3 */}
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Database Engine</p>
                  <p className="text-sm font-mono">{instance.database_engine}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Authentication</p>
                  <p className="text-sm">
                    {instance.use_rds_iam_auth ? 'RDS IAM Authentication' : 'Password'}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Created</p>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Calendar className="h-4 w-4" />
                    <span>{formatDate(instance.created_at)}</span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        ) : null}

        {/* Database Sync Results */}
        {syncResult && (
          <Card className="border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950">
            <CardContent className="pt-6">
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
                    <h3 className="font-semibold text-green-800 dark:text-green-200">
                      Database Sync Completed
                    </h3>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setSyncResult(null)}
                    className="text-green-700 hover:text-green-900 dark:text-green-300 dark:hover:text-green-100"
                  >
                    Dismiss
                  </Button>
                </div>
                <div className="grid grid-cols-4 gap-4 text-sm">
                  <div>
                    <span className="text-green-700 dark:text-green-300">Total Found:</span>{' '}
                    <span className="font-medium text-green-900 dark:text-green-100">
                      {syncResult.total_found}
                    </span>
                  </div>
                  <div>
                    <span className="text-green-700 dark:text-green-300">Synced:</span>{' '}
                    <span className="font-medium text-green-900 dark:text-green-100">
                      {syncResult.synced}
                    </span>
                  </div>
                  <div>
                    <span className="text-green-700 dark:text-green-300">Skipped:</span>{' '}
                    <span className="font-medium text-green-900 dark:text-green-100">
                      {syncResult.skipped}
                    </span>
                  </div>
                  <div>
                    <span className="text-green-700 dark:text-green-300">Failed:</span>{' '}
                    <span className="font-medium text-green-900 dark:text-green-100">
                      {syncResult.failed}
                    </span>
                  </div>
                </div>
                {syncResult.skipped_databases && syncResult.skipped_databases.length > 0 && (
                  <div className="text-sm">
                    <span className="text-green-700 dark:text-green-300">Skipped databases:</span>{' '}
                    <span className="text-green-800 dark:text-green-200">
                      {syncResult.skipped_databases.join(', ')}
                    </span>
                  </div>
                )}
                {syncResult.failed_databases && syncResult.failed_databases.length > 0 && (
                  <div className="text-sm">
                    <span className="text-red-700 dark:text-red-300">Failed databases:</span>{' '}
                    <span className="text-red-800 dark:text-red-200">
                      {syncResult.failed_databases.join(', ')}
                    </span>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Databases Table */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Databases</CardTitle>
                <CardDescription>
                  {isLoadingDatabases ? (
                    'Loading databases...'
                  ) : (
                    <>
                      {databases.length} database{databases.length === 1 ? '' : 's'} using this
                      instance
                    </>
                  )}
                </CardDescription>
              </div>
              {canCreateDatabases && instance && (
                <Button onClick={() => setShowCreateDialog(true)} size="sm">
                  <Plus className="mr-2 h-4 w-4" />
                  Create Database
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {!isLoadingDatabases && databases.length === 0 ? (
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <DatabaseIcon className="h-12 w-12 text-muted-foreground" />
                  </EmptyMedia>
                  <EmptyTitle>No Databases Found</EmptyTitle>
                  <EmptyDescription>
                    No databases are currently using this instance.
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  <Button onClick={() => navigate('/databases')}>
                    <DatabaseIcon className="mr-2 h-4 w-4" />
                    View All Databases
                  </Button>
                </EmptyContent>
              </Empty>
            ) : (
              <DataTable
                columns={databasesColumns}
                data={databases}
                isLoading={isLoadingDatabases}
                searchable
                searchKey="database_name"
                searchPlaceholder="Search by database name..."
                emptyMessage="No databases found."
                pagination={{
                  pageIndex,
                  pageSize,
                  totalCount: databases.length,
                  onPageChange: setPageIndex,
                  onPageSizeChange: (newSize) => {
                    setPageSize(newSize);
                    setPageIndex(0);
                  },
                  pageSizeOptions: [10, 20, 30, 50],
                }}
                onRowClick={(database) => navigate(`/databases/${database.id}`)}
              />
            )}
          </CardContent>
        </Card>
      </div>

      {/* Create Database Dialog */}
      {id && (
        <CreateDatabaseDialog
          open={showCreateDialog}
          onOpenChange={setShowCreateDialog}
          databaseInstanceId={id}
        />
      )}
    </DefaultLayout>
  );
};
