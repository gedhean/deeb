import React, { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { subDays } from 'date-fns';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import {
  useDatabaseInstancesSWR,
  useCreateDatabaseInstance,
  useUpdateDatabaseInstance,
  useDeleteDatabaseInstance,
  useTestConnection,
} from '@/hooks/api/useDatabaseInstancesSWR';
import { useEnvironmentsSWR } from '@/hooks/api/useEnvironmentsSWR';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';
import { objDiff } from '@/lib/utils';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import {
  DatabaseInstanceForm,
  type DatabaseInstanceFormData,
  type DatabaseInstanceFormErrors,
} from '@/components/DatabaseInstanceForm';
import type {
  CreateDatabaseInstanceRequest,
  DatabaseInstance,
  UpdateDatabaseInstanceRequest,
  DatabaseEngine,
} from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { DataTable, DataTableToolbar, DataTableViewOptions } from '@/components/data-table';
import type { FilterDefinition } from '@/components/data-table/types';
import { getDatabaseInstancesColumns } from './database-instances/columns';
import { Plus, AlertTriangle, CheckCircle2, Loader2, Server } from 'lucide-react';

export const DatabaseInstancesPage: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedInstance, setSelectedInstance] = useState<DatabaseInstance | null>(null);
  const [connectionTestResult, setConnectionTestResult] = useState<{
    success: boolean;
    message?: string;
  } | null>(null);
  const [syncResult, setSyncResult] = useState<{
    total_found: number;
    synced: number;
    failed: number;
    skipped: number;
    failed_databases?: string[];
    skipped_databases?: string[];
  } | null>(null);

  // Use SWR hooks for data fetching
  const {
    databaseInstances: instances,
    isLoading,
    error: instancesError,
  } = useDatabaseInstancesSWR();
  const { environments } = useEnvironmentsSWR();

  // Use mutation hooks
  const { createDatabaseInstance, isLoading: isCreating } = useCreateDatabaseInstance();
  const { updateDatabaseInstance, isLoading: isUpdating } = useUpdateDatabaseInstance();
  const { deleteDatabaseInstance, isLoading: isDeleting } = useDeleteDatabaseInstance();
  const { testConnection, isLoading: isTestingConnection } = useTestConnection();

  // Pagination state
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(10);

  // Form state
  const [formData, setFormData] = useState<DatabaseInstanceFormData>({
    name: '',
    description: '',
    database_engine: 'postgres' as DatabaseEngine,
    host: '',
    port: 5432,
    username: '',
    password: '',
    use_rds_iam_auth: false,
    require_ssl: false,
    environment_id: undefined,
    sync_all_databases: false,
  });
  const [formErrors, setFormErrors] = useState<DatabaseInstanceFormErrors>({});

  // Check permissions - Database instances are managed by both Admin and DBA
  const canReadDatabaseInstances = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabaseInstancesRead,
  );
  const canCreateDatabaseInstances = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabaseInstancesCreate,
  );
  const canUpdateDatabaseInstances = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabaseInstancesUpdate,
  );
  const canDeleteDatabaseInstances = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabaseInstancesDelete,
  );

  // Compute error message
  const error = !canReadDatabaseInstances
    ? 'Access denied. You do not have permission to view database instances.'
    : instancesError || '';

  // Handle database engine change (for auto-updating port in create mode)
  const handleDatabaseEngineChange = (engine: DatabaseEngine) => {
    setFormData({
      ...formData,
      database_engine: engine,
      port: engine === 'mysql' ? 3306 : 5432,
    });
  };

  // Validate form
  const validateForm = (): boolean => {
    const errors: DatabaseInstanceFormErrors = {};

    if (!formData.name.trim()) {
      errors.name = 'Instance name is required';
    }

    if (!formData.host.trim()) {
      errors.host = 'Host is required';
    }

    if (!formData.port || formData.port < 1 || formData.port > 65535) {
      errors.port = 'Port must be between 1 and 65535';
    }

    if (!formData.username.trim()) {
      errors.username = 'Username is required';
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Test connection
  const handleTestConnection = async () => {
    if (!validateForm()) {
      return;
    }

    setConnectionTestResult(null);

    let result;

    // If we're editing an existing instance, use the dedicated endpoint
    if (selectedInstance?.id) {
      result = await databaseInstancesService.testConnection(selectedInstance.id);
    } else {
      // For new instances, test with credentials
      // Only send the fields needed for connection testing
      const testData: CreateDatabaseInstanceRequest = {
        host: formData.host,
        port: formData.port,
        username: formData.username,
        password: formData.password,
        database_engine: formData.database_engine,
        use_rds_iam_auth: formData.use_rds_iam_auth,
        require_ssl: formData.require_ssl,
        name: '', // Not used for testing, but required by type
      };
      result = await testConnection(testData);
    }

    if (result.success) {
      setConnectionTestResult({
        success: true,
        message: `Connection successful to ${result.username}@${result.host}:${result.port}/${result.database}.`,
      });
    } else {
      setConnectionTestResult({
        success: false,
        message: result.error || 'Connection test failed',
      });
    }
  };

  // Handle create
  const handleCreate = async () => {
    if (!validateForm()) {
      return;
    }

    const createData: CreateDatabaseInstanceRequest = {
      name: formData.name.trim(),
      description: formData.description?.trim() || undefined,
      database_engine: formData.database_engine,
      host: formData.host.trim(),
      port: formData.port,
      username: formData.username.trim(),
      password: formData.password,
      use_rds_iam_auth: formData.use_rds_iam_auth,
      require_ssl: formData.require_ssl,
      environment_id: formData.environment_id,
      sync_all_databases: formData.sync_all_databases,
    };

    const result = await createDatabaseInstance(createData);
    if (result) {
      // Store sync results if present
      if (result.sync_results) {
        setSyncResult(result.sync_results);
      }

      setIsCreateOpen(false);
      setFormData({
        name: '',
        description: '',
        database_engine: 'postgres' as DatabaseEngine,
        host: '',
        port: 5432,
        username: '',
        password: '',
        use_rds_iam_auth: false,
        require_ssl: false,
        environment_id: undefined,
        sync_all_databases: false,
      });
      setFormErrors({});
      setConnectionTestResult(null);
    }
  };

  // Handle form data change
  const handleFormDataChange = (data: Partial<DatabaseInstanceFormData>) => {
    setFormData((prev) => ({ ...prev, ...data }));
  };

  // Handle edit
  const handleEdit = async () => {
    if (!selectedInstance || !validateForm()) {
      return;
    }

    // Normalize form data (trim strings, handle optional fields)
    const normalizedFormData = {
      name: formData.name.trim(),
      description: formData.description?.trim() || undefined,
      database_engine: formData.database_engine,
      host: formData.host.trim(),
      port: formData.port,
      username: formData.username.trim(),
      use_rds_iam_auth: formData.use_rds_iam_auth,
      require_ssl: formData.require_ssl,
      environment_id: formData.environment_id || undefined,
    };

    // Get only the fields that changed
    const updateData: UpdateDatabaseInstanceRequest = objDiff(selectedInstance, normalizedFormData);

    // Special handling for password - only include if user entered a new one
    if (formData.password) {
      updateData.password = formData.password;
    }

    const result = await updateDatabaseInstance(selectedInstance.id, updateData);
    if (result) {
      setIsEditOpen(false);
      setSelectedInstance(null);
      setFormData({
        name: '',
        description: '',
        database_engine: 'postgres' as DatabaseEngine,
        host: '',
        port: 5432,
        username: '',
        password: '',
        use_rds_iam_auth: false,
        require_ssl: false,
        environment_id: undefined,
      });
      setFormErrors({});
      setConnectionTestResult(null);
    }
  };

  // Handle delete
  const handleDelete = async () => {
    if (!selectedInstance) {
      return;
    }

    const success = await deleteDatabaseInstance(selectedInstance.id);
    if (success) {
      setIsDeleteOpen(false);
      setSelectedInstance(null);
    }
  };

  // Open create form
  const openCreateForm = () => {
    setSelectedInstance(null); // Clear selected instance for new creation
    setFormData({
      name: '',
      description: '',
      database_engine: 'postgres' as DatabaseEngine,
      host: '',
      port: 5432,
      username: '',
      password: '',
      use_rds_iam_auth: false,
      require_ssl: false,
    });
    setFormErrors({});
    setConnectionTestResult(null);
    setIsCreateOpen(true);
  };

  // Open edit form
  const openEditForm = useCallback((instance: DatabaseInstance) => {
    setSelectedInstance(instance);
    setFormData({
      name: instance.name,
      description: instance.description || '',
      database_engine: instance.database_engine,
      host: instance.host,
      port: instance.port,
      username: instance.username,
      password: '', // Never show existing password
      use_rds_iam_auth: instance.use_rds_iam_auth,
      require_ssl: instance.require_ssl,
      environment_id: instance.environment_id || undefined,
    });
    setFormErrors({});
    setConnectionTestResult(null);
    setIsEditOpen(true);
  }, []);

  // Close edit form
  const closeEditForm = () => {
    setIsEditOpen(false);
    setSelectedInstance(null); // Clear selected instance
  };

  // Open delete confirmation
  const openDeleteConfirmation = useCallback((instance: DatabaseInstance) => {
    setSelectedInstance(instance);
    setIsDeleteOpen(true);
  }, []);

  // Helper to get environment name
  const getEnvironmentName = useCallback(
    (environmentId?: string): string => {
      if (!environmentId) return '-';
      return environments.find((env) => env.id === environmentId)?.name || 'Unknown';
    },
    [environments],
  );

  // Create filter options
  const environmentOptions = useMemo(
    () =>
      environments.map((env) => ({
        label: env.name,
        value: env.id,
      })),
    [environments],
  );

  // Get columns with action handlers
  const columns = useMemo(
    () =>
      getDatabaseInstancesColumns({
        onView: (instance) => navigate(`/database-instances/${instance.id}`),
        onEdit: openEditForm,
        onDelete: openDeleteConfirmation,
        canUpdate: canUpdateDatabaseInstances,
        canDelete: canDeleteDatabaseInstances,
        getEnvironmentName,
      }),
    [
      navigate,
      openEditForm,
      openDeleteConfirmation,
      canUpdateDatabaseInstances,
      canDeleteDatabaseInstances,
      getEnvironmentName,
    ],
  );

  // Define filter definitions for the toolbar
  const filterDefinitions: FilterDefinition[] = useMemo(
    () => [
      {
        id: 'name',
        type: 'search',
        title: 'Name',
        placeholder: 'Search names...',
        defaultActive: true,
      },
      {
        id: 'database_engine',
        type: 'faceted',
        title: 'Engine',
        options: [
          { label: 'PostgreSQL', value: 'postgres', icon: Server },
          { label: 'MySQL', value: 'mysql', icon: Server },
        ],
      },
      {
        id: 'environment',
        type: 'faceted',
        title: 'Environment',
        options: environmentOptions,
      },
      {
        id: 'status',
        type: 'faceted',
        title: 'Status',
        options: [
          { label: 'Active', value: 'Active', icon: Server },
          { label: 'Inactive', value: 'Inactive', icon: Server },
        ],
      },
      {
        id: 'created_at',
        type: 'dateRange',
        title: 'Created',
        presets: [
          {
            label: 'Last 7 days',
            getValue: () => [subDays(new Date(), 7), new Date()],
          },
          {
            label: 'Last 30 days',
            getValue: () => [subDays(new Date(), 30), new Date()],
          },
          {
            label: 'Last 90 days',
            getValue: () => [subDays(new Date(), 90), new Date()],
          },
        ],
      },
    ],
    [environmentOptions],
  );

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
              This page requires database read permissions. Please contact your administrator if you
              need access.
            </p>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Database Instances', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Database Instances</h1>
            <p className="text-muted-foreground mt-1">
              Manage database server connections and credentials
            </p>
          </div>
          {canCreateDatabaseInstances && (
            <Button onClick={openCreateForm}>
              <Plus className="mr-2 h-4 w-4" />
              Create Instance
            </Button>
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

        {/* Instances list */}
        <Card>
          <CardContent className="pt-6">
            <DataTable
              columns={columns}
              data={instances}
              isLoading={isLoading}
              emptyMessage="No database instances match your filters. Try adjusting your search criteria."
              pagination={{
                pageIndex,
                pageSize,
                totalCount: instances.length,
                onPageChange: setPageIndex,
                onPageSizeChange: (newSize) => {
                  setPageSize(newSize);
                  setPageIndex(0);
                },
                pageSizeOptions: [10, 20, 30, 50],
              }}
              onRowClick={(instance) => navigate(`/database-instances/${instance.id}`)}
            >
              {(table) => (
                <DataTableToolbar table={table} filters={filterDefinitions}>
                  <DataTableViewOptions table={table} />
                </DataTableToolbar>
              )}
            </DataTable>
          </CardContent>
        </Card>

        {/* Create Instance Sheet */}
        <Sheet open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <SheetContent className="w-full min-w-[600px] sm:w-[540px] overflow-y-auto max-h-[90vh]">
            <SheetHeader>
              <SheetTitle>Create Database Instance</SheetTitle>
              <SheetDescription>
                Create a new database instance connection. Connection will be tested before saving.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 mb-6">
              <DatabaseInstanceForm
                mode="create"
                formData={formData}
                formErrors={formErrors}
                environments={environments}
                onChange={handleFormDataChange}
                onDatabaseEngineChange={handleDatabaseEngineChange}
              />

              {/* Connection Test Result */}
              {connectionTestResult && (
                <div
                  className={`rounded-md border p-3 ${
                    connectionTestResult.success
                      ? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950'
                      : 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {connectionTestResult.success ? (
                      <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                    ) : (
                      <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400" />
                    )}
                    <span
                      className={`text-sm ${
                        connectionTestResult.success
                          ? 'text-green-800 dark:text-green-200'
                          : 'text-red-800 dark:text-red-200'
                      }`}
                    >
                      {connectionTestResult.message}
                    </span>
                  </div>
                </div>
              )}

              {/* Connection Test Button */}
              <Button
                type="button"
                variant="outline"
                onClick={handleTestConnection}
                disabled={isTestingConnection || isCreating}
                className="w-full"
              >
                {isTestingConnection ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Testing Connection...
                  </>
                ) : (
                  'Test Connection'
                )}
              </Button>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={isCreating || isTestingConnection}>
                {isCreating ? 'Creating...' : 'Create Instance'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Edit Instance Sheet */}
        <Sheet open={isEditOpen} onOpenChange={setIsEditOpen}>
          <SheetContent className="overflow-y-auto max-h-[90vh] min-w-[600px] sm:w-[540px]">
            <SheetHeader>
              <SheetTitle>Edit Database Instance</SheetTitle>
              <SheetDescription>
                Update database instance connection. Connection will be tested if credentials
                change.
              </SheetDescription>
            </SheetHeader>
            <div className="space-y-4 mb-6">
              <DatabaseInstanceForm
                mode="edit"
                formData={formData}
                formErrors={formErrors}
                environments={environments}
                onChange={handleFormDataChange}
              />

              {/* Connection Test Result */}
              {connectionTestResult && (
                <div
                  className={`rounded-md border p-3 ${
                    connectionTestResult.success
                      ? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950'
                      : 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    {connectionTestResult.success ? (
                      <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                    ) : (
                      <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400" />
                    )}
                    <span
                      className={`text-sm ${
                        connectionTestResult.success
                          ? 'text-green-800 dark:text-green-200'
                          : 'text-red-800 dark:text-red-200'
                      }`}
                    >
                      {connectionTestResult.message}
                    </span>
                  </div>
                </div>
              )}

              {/* Connection Test Button */}
              <Button
                type="button"
                variant="outline"
                onClick={handleTestConnection}
                disabled={isTestingConnection || isUpdating}
                className="w-full"
              >
                {isTestingConnection ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Testing Connection...
                  </>
                ) : (
                  'Test Connection'
                )}
              </Button>
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={closeEditForm}>
                Cancel
              </Button>
              <Button onClick={handleEdit} disabled={isUpdating || isTestingConnection}>
                {isUpdating ? 'Saving...' : 'Save Changes'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>

        {/* Delete Confirmation Sheet */}
        <Sheet open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
          <SheetContent>
            <SheetHeader>
              <SheetTitle>Delete Database Instance</SheetTitle>
              <SheetDescription>
                Are you sure you want to delete this database instance? This action cannot be
                undone.
              </SheetDescription>
            </SheetHeader>
            <div className="py-4">
              {selectedInstance && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">{selectedInstance.name}</p>
                  <p className="text-sm text-muted-foreground">
                    {selectedInstance.description || 'No description'}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {selectedInstance.host}:{selectedInstance.port}
                  </p>
                </div>
              )}
            </div>
            <SheetFooter>
              <Button variant="outline" onClick={() => setIsDeleteOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleDelete} disabled={isDeleting}>
                {isDeleting ? 'Deleting...' : 'Delete Instance'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
