import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import { environmentsService } from '@/lib/api/services/environments';
import { hasOrganizationPermission, OrganizationPermission } from '@/lib/rbac';
import { DatabaseStatusBadge } from '@/components/DatabaseStatusBadge';
import {
  DatabaseInstanceForm,
  type DatabaseInstanceFormData,
  type DatabaseInstanceFormErrors,
} from '@/components/DatabaseInstanceForm';
import type {
  CreateDatabaseInstanceRequest,
  DatabaseInstance,
  UpdateDatabaseInstanceRequest,
  Environment,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Plus,
  Pencil,
  Trash2,
  AlertTriangle,
  CheckCircle2,
  Loader2,
  Server,
  Search,
  X,
  Eye,
} from 'lucide-react';
import type { DatabaseEngine } from '@/lib/api/types.ts';

export const DatabaseInstancesPage: React.FC = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [instances, setInstances] = useState<DatabaseInstance[]>([]);
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [selectedInstance, setSelectedInstance] = useState<DatabaseInstance | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [connectionTestResult, setConnectionTestResult] = useState<{
    success: boolean;
    message?: string;
  } | null>(null);

  // Filter state
  const [filters, setFilters] = useState<{
    search?: string;
    database_engine?: DatabaseEngine;
    status?: string;
    environment_id?: string;
  }>({});

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

  // Load instances and environments
  useEffect(() => {
    if (!canReadDatabaseInstances) {
      setError('Access denied. You do not have permission to view database instances.');
      setIsLoading(false);
      return;
    }

    loadData();
  }, [canReadDatabaseInstances]);

  const loadData = async () => {
    try {
      setIsLoading(true);
      setError('');
      const [instancesResponse, environmentsResponse] = await Promise.all([
        databaseInstancesService.listDatabaseInstances({ limit: 100 }),
        environmentsService.listEnvironments({ limit: 100 }),
      ]);
      setInstances(instancesResponse.items);
      setEnvironments(environmentsResponse.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load data. Please try again.');
      }
      console.error('Failed to load data:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const loadInstances = async () => {
    try {
      setError('');
      const response = await databaseInstancesService.listDatabaseInstances({ limit: 100 });
      setInstances(response.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load database instances. Please try again.');
      }
      console.error('Failed to load database instances:', err);
    }
  };

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

    try {
      setIsTestingConnection(true);
      setConnectionTestResult(null);

      // Create a test request with a temporary name to avoid conflicts
      const testData: CreateDatabaseInstanceRequest = {
        ...formData,
        name: `__TEST__${formData.name}_${Date.now()}`,
      };

      // Test connection by attempting to create
      // Backend will test connection and return error if it fails
      const result = await databaseInstancesService.testConnection(testData);

      if (result.success) {
        // Connection was successful
        // If we got an instance back, delete it immediately to clean up
        if (result.instance) {
          try {
            await databaseInstancesService.deleteDatabaseInstance(result.instance.id);
            setConnectionTestResult({
              success: true,
              message: 'Connection test successful!',
            });
          } catch {
            // If deletion fails, just show success with a note
            setConnectionTestResult({
              success: true,
              message:
                'Connection test successful! Note: A test instance was created (you can delete it).',
            });
            await loadInstances();
          }
        } else {
          setConnectionTestResult({
            success: true,
            message: 'Connection test successful!',
          });
        }
      } else {
        setConnectionTestResult({
          success: false,
          message: result.details || result.error || 'Connection test failed',
        });
      }
    } catch (err) {
      setConnectionTestResult({
        success: false,
        message: err instanceof Error ? err.message : 'Connection test failed',
      });
    } finally {
      setIsTestingConnection(false);
    }
  };

  // Handle create
  const handleCreate = async () => {
    if (!validateForm()) {
      return;
    }

    try {
      setIsSubmitting(true);
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
      };
      await databaseInstancesService.createDatabaseInstance(createData);
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
      });
      setFormErrors({});
      setConnectionTestResult(null);
      await loadInstances();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to create database instance. Please try again.');
      }
      console.error('Failed to create database instance:', err);
    } finally {
      setIsSubmitting(false);
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

    try {
      setIsSubmitting(true);
      const updateData: UpdateDatabaseInstanceRequest = {};
      if (formData.name !== selectedInstance.name) {
        updateData.name = formData.name.trim();
      }
      if (formData.description !== selectedInstance.description) {
        updateData.description = formData.description?.trim() || undefined;
      }
      if (formData.database_engine !== selectedInstance.database_engine) {
        updateData.database_engine = formData.database_engine;
      }
      if (formData.host !== selectedInstance.host) {
        updateData.host = formData.host.trim();
      }
      if (formData.port !== selectedInstance.port) {
        updateData.port = formData.port;
      }
      if (formData.username !== selectedInstance.username) {
        updateData.username = formData.username.trim();
      }
      if (formData.password) {
        updateData.password = formData.password;
      }
      if (formData.use_rds_iam_auth !== selectedInstance.use_rds_iam_auth) {
        updateData.use_rds_iam_auth = formData.use_rds_iam_auth;
      }
      if (formData.require_ssl !== selectedInstance.require_ssl) {
        updateData.require_ssl = formData.require_ssl;
      }
      if (formData.environment_id !== selectedInstance.environment_id) {
        updateData.environment_id = formData.environment_id || undefined;
      }

      await databaseInstancesService.updateDatabaseInstance(selectedInstance.id, updateData);
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
      await loadInstances();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to update database instance. Please try again.');
      }
      console.error('Failed to update database instance:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle delete
  const handleDelete = async () => {
    if (!selectedInstance) {
      return;
    }

    try {
      setIsSubmitting(true);
      await databaseInstancesService.deleteDatabaseInstance(selectedInstance.id);
      setIsDeleteOpen(false);
      setSelectedInstance(null);
      await loadInstances();
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to delete database instance. Please try again.');
      }
      console.error('Failed to delete database instance:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Open create form
  const openCreateForm = () => {
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
  const openEditForm = (instance: DatabaseInstance) => {
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
  };

  // Open delete confirmation
  const openDeleteConfirmation = (instance: DatabaseInstance) => {
    setSelectedInstance(instance);
    setIsDeleteOpen(true);
  };

  // Format date
  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      });
    } catch {
      return dateString;
    }
  };

  // Handle filter changes
  const handleFilterChange = (key: keyof typeof filters, value: string | undefined) => {
    setFilters((prev) => {
      const updated = { ...prev };
      if (value && value.trim()) {
        updated[key] = value as unknown as DatabaseEngine;
      } else {
        delete updated[key];
      }
      return updated;
    });
  };

  // Clear all filters
  const clearFilters = () => {
    setFilters({});
  };

  // Filter instances
  const filteredInstances = instances.filter((instance) => {
    // Search filter (name and host)
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      const matchesSearch =
        instance.name.toLowerCase().includes(searchLower) ||
        instance.host.toLowerCase().includes(searchLower) ||
        instance.username.toLowerCase().includes(searchLower) ||
        (instance.description && instance.description.toLowerCase().includes(searchLower));
      if (!matchesSearch) return false;
    }

    // Database type filter
    if (filters.database_engine && instance.database_engine !== filters.database_engine) {
      return false;
    }

    // Status filter
    if (filters.status && instance.status !== filters.status) {
      return false;
    }

    // Environment filter
    if (filters.environment_id && instance.environment_id !== filters.environment_id) {
      return false;
    }

    return true;
  });

  const hasFilters = Object.keys(filters).length > 0;

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

        {/* Filters */}
        <Card>
          <CardHeader>
            <CardTitle>Filters</CardTitle>
            <CardDescription>
              Filter database instances by name, host, type, or status
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {/* Search Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-search" className="text-sm font-medium">
                  Search
                </label>
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    id="filter-search"
                    placeholder="Search instances..."
                    value={filters.search || ''}
                    onChange={(e) => handleFilterChange('search', e.target.value)}
                    className="pl-9"
                  />
                </div>
              </div>

              {/* Database Engine Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-type" className="text-sm font-medium">
                  Database Engine
                </label>
                <Select
                  value={filters.database_engine || 'all'}
                  onValueChange={(value) =>
                    handleFilterChange('database_engine', value === 'all' ? undefined : value)
                  }
                >
                  <SelectTrigger id="filter-type">
                    <SelectValue placeholder="All Types" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Types</SelectItem>
                    <SelectItem value="postgres">PostgreSQL</SelectItem>
                    <SelectItem value="mysql">MySQL</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Status Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-status" className="text-sm font-medium">
                  Status
                </label>
                <Select
                  value={filters.status || 'all'}
                  onValueChange={(value) =>
                    handleFilterChange('status', value === 'all' ? undefined : value)
                  }
                >
                  <SelectTrigger id="filter-status">
                    <SelectValue placeholder="All Statuses" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Statuses</SelectItem>
                    <SelectItem value="Active">Active</SelectItem>
                    <SelectItem value="Inactive">Inactive</SelectItem>
                    <SelectItem value="Maintenance">Maintenance</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Environment Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-environment" className="text-sm font-medium">
                  Environment
                </label>
                <Select
                  value={filters.environment_id || 'all'}
                  onValueChange={(value) =>
                    handleFilterChange('environment_id', value === 'all' ? undefined : value)
                  }
                >
                  <SelectTrigger id="filter-environment">
                    <SelectValue placeholder="All Environments" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Environments</SelectItem>
                    {environments.map((env) => (
                      <SelectItem key={env.id} value={env.id}>
                        {env.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Clear Filters Button */}
            {hasFilters && (
              <div className="mt-4">
                <Button variant="outline" size="sm" onClick={clearFilters}>
                  <X className="mr-2 h-4 w-4" />
                  Clear Filters
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Instances list */}
        {isLoading ? (
          <Card>
            <CardContent className="pt-6">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Host</TableHead>
                    <TableHead>Username</TableHead>
                    <TableHead>Status</TableHead>
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
                        <Skeleton className="h-4 w-40" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-24" />
                      </TableCell>
                      <TableCell>
                        <Skeleton className="h-4 w-20" />
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
            </CardContent>
          </Card>
        ) : filteredInstances.length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Server className="h-12 w-12 text-muted-foreground" />
                  </EmptyMedia>
                  <EmptyTitle>
                    {hasFilters ? 'No Instances Found' : 'No Database Instances Yet'}
                  </EmptyTitle>
                  <EmptyDescription>
                    {hasFilters
                      ? 'No database instances match your filters. Try adjusting your search criteria.'
                      : 'Get started by creating your first database instance connection to manage server credentials.'}
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  {hasFilters ? (
                    <Button variant="outline" onClick={clearFilters}>
                      <X className="mr-2 h-4 w-4" />
                      Clear Filters
                    </Button>
                  ) : canCreateDatabaseInstances ? (
                    <Button onClick={openCreateForm}>
                      <Plus className="mr-2 h-4 w-4" />
                      Create Instance
                    </Button>
                  ) : null}
                </EmptyContent>
              </Empty>
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>Database Instances</CardTitle>
              <CardDescription>
                {filteredInstances.length} of {instances.length} instance
                {instances.length === 1 ? '' : 's'}
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-6">
              <div className="overflow-x-auto -mx-6 px-6">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Host</TableHead>
                      <TableHead>Username</TableHead>
                      <TableHead>Environment</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredInstances.map((instance) => (
                      <TableRow
                        key={instance.id}
                        onClick={() => navigate(`/database-instances/${instance.id}`)}
                        className="cursor-pointer hover:bg-muted/50 transition-colors"
                      >
                        <TableCell className="font-medium">{instance.name}</TableCell>
                        <TableCell className="text-muted-foreground">
                          {instance.database_engine}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {instance.host}:{instance.port}
                        </TableCell>
                        <TableCell className="text-muted-foreground">{instance.username}</TableCell>
                        <TableCell className="text-muted-foreground">
                          {instance.environment_id
                            ? environments.find((env) => env.id === instance.environment_id)
                                ?.name || 'Unknown'
                            : '-'}
                        </TableCell>
                        <TableCell>
                          <DatabaseStatusBadge status={instance.status} variant="instance" />
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(instance.created_at)}
                        </TableCell>
                        <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                          <div className="flex items-center justify-end gap-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => navigate(`/database-instances/${instance.id}`)}
                              className="h-11 w-11 p-0 touch-manipulation"
                              aria-label="View instance"
                            >
                              <Eye className="h-4 w-4" />
                            </Button>
                            {canUpdateDatabaseInstances && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => openEditForm(instance)}
                                className="h-11 w-11 p-0 touch-manipulation"
                                aria-label="Edit instance"
                              >
                                <Pencil className="h-4 w-4" />
                              </Button>
                            )}
                            {canDeleteDatabaseInstances && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => openDeleteConfirmation(instance)}
                                className="h-11 w-11 p-0 text-destructive hover:text-destructive touch-manipulation"
                                aria-label="Delete instance"
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        )}

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
                disabled={isTestingConnection || isSubmitting}
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
              <Button onClick={handleCreate} disabled={isSubmitting || isTestingConnection}>
                {isSubmitting ? 'Creating...' : 'Create Instance'}
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
                disabled={isTestingConnection || isSubmitting}
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
              <Button variant="outline" onClick={() => setIsEditOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleEdit} disabled={isSubmitting || isTestingConnection}>
                {isSubmitting ? 'Saving...' : 'Save Changes'}
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
              <Button variant="destructive" onClick={handleDelete} disabled={isSubmitting}>
                {isSubmitting ? 'Deleting...' : 'Delete Instance'}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
