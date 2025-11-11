import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { useAuth } from '@/contexts/AuthContext';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import { environmentsService } from '@/lib/api/services/environments';
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
    database_engine?: string;
    status?: string;
    environment_id?: string;
  }>({});

  // Form state
  const [formData, setFormData] = useState<CreateDatabaseInstanceRequest>({
    name: '',
    description: '',
    database_engine: 'postgres',
    host: '',
    port: 5432,
    username: '',
    password: '',
    use_rds_iam_auth: false,
    require_ssl: false,
    environment_id: undefined,
  });
  const [formErrors, setFormErrors] = useState<{
    name?: string;
    host?: string;
    port?: string;
    username?: string;
    password?: string;
  }>({});

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  // Load instances and environments
  useEffect(() => {
    if (!isAdmin) {
      setError('Access denied. Admin access required.');
      setIsLoading(false);
      return;
    }

    loadData();
  }, [isAdmin]);

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

  // Validate form
  const validateForm = (): boolean => {
    const errors: {
      name?: string;
      host?: string;
      port?: string;
      username?: string;
      password?: string;
    } = {};

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
        environment_id: formData.environment_id || undefined,
      };
      await databaseInstancesService.createDatabaseInstance(createData);
      setIsCreateOpen(false);
      setFormData({
        name: '',
        description: '',
        database_engine: 'postgres',
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
        database_engine: 'postgres',
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
      database_engine: 'postgres',
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

  // Get status badge color
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Active':
        return 'text-green-600 dark:text-green-400';
      case 'Inactive':
        return 'text-gray-600 dark:text-gray-400';
      case 'Maintenance':
        return 'text-yellow-600 dark:text-yellow-400';
      default:
        return 'text-gray-600 dark:text-gray-400';
    }
  };

  // Handle filter changes
  const handleFilterChange = (key: keyof typeof filters, value: string | undefined) => {
    setFilters((prev) => {
      const updated = { ...prev };
      if (value && value.trim()) {
        updated[key] = value;
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

    return true;
  });

  const hasFilters = Object.keys(filters).length > 0;

  // Render access denied
  if (!isAdmin) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Database Instances', isCurrentPage: true }]}>
        <Card>
          <CardHeader>
            <CardTitle>Access Denied</CardTitle>
            <CardDescription>You do not have permission to access this page.</CardDescription>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              This page requires Organization Admin role.
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
          <Button onClick={openCreateForm}>
            <Plus className="mr-2 h-4 w-4" />
            Create Instance
          </Button>
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
                <select
                  id="filter-type"
                  value={filters.database_engine || ''}
                  onChange={(e) => handleFilterChange('database_engine', e.target.value)}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">All Types</option>
                  <option value="postgresql">PostgreSQL</option>
                  <option value="mysql">MySQL</option>
                </select>
              </div>

              {/* Status Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-status" className="text-sm font-medium">
                  Status
                </label>
                <select
                  id="filter-status"
                  value={filters.status || ''}
                  onChange={(e) => handleFilterChange('status', e.target.value)}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">All Statuses</option>
                  <option value="Active">Active</option>
                  <option value="Inactive">Inactive</option>
                  <option value="Maintenance">Maintenance</option>
                </select>
              </div>

              {/* Environment Filter */}
              <div className="space-y-2">
                <label htmlFor="filter-environment" className="text-sm font-medium">
                  Environment
                </label>
                <select
                  id="filter-environment"
                  value={filters.environment_id || ''}
                  onChange={(e) => handleFilterChange('environment_id', e.target.value)}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">All Environments</option>
                  {environments.map((env) => (
                    <option key={env.id} value={env.id}>
                      {env.name}
                    </option>
                  ))}
                </select>
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
                  ) : (
                    <Button onClick={openCreateForm}>
                      <Plus className="mr-2 h-4 w-4" />
                      Create Instance
                    </Button>
                  )}
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
                          <span className={getStatusColor(instance.status)}>{instance.status}</span>
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
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openEditForm(instance)}
                              className="h-11 w-11 p-0 touch-manipulation"
                              aria-label="Edit instance"
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openDeleteConfirmation(instance)}
                              className="h-11 w-11 p-0 text-destructive hover:text-destructive touch-manipulation"
                              aria-label="Delete instance"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
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
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="create-name" className="text-sm font-medium">
                  Instance Name *
                </label>
                <Input
                  id="create-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter instance name"
                  aria-invalid={!!formErrors.name}
                />
                {formErrors.name && <p className="text-sm text-destructive">{formErrors.name}</p>}
              </div>
              <div className="space-y-2">
                <label htmlFor="create-description" className="text-sm font-medium">
                  Description
                </label>
                <textarea
                  id="create-description"
                  value={formData.description || ''}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Enter instance description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="create-type" className="text-sm font-medium">
                  Database Engine *
                </label>
                <select
                  id="create-type"
                  value={formData.database_engine}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      database_engine: e.target.value as 'mysql' | 'postgres',
                      port: e.target.value === 'mysql' ? 3306 : 5432,
                    })
                  }
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="postgresql">PostgreSQL</option>
                  <option value="mysql">MySQL</option>
                </select>
              </div>
              <div className="space-y-2">
                <label htmlFor="create-environment" className="text-sm font-medium">
                  Environment (optional)
                </label>
                <select
                  id="create-environment"
                  value={formData.environment_id || ''}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      environment_id: e.target.value || undefined,
                    })
                  }
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">No Environment</option>
                  {environments.map((env) => (
                    <option key={env.id} value={env.id}>
                      {env.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-4 gap-4">
                <div className="space-y-2 col-span-3">
                  <label htmlFor="create-host" className="text-sm font-medium">
                    Host *
                  </label>
                  <Input
                    id="create-host"
                    value={formData.host}
                    onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                    placeholder="Enter database host"
                    aria-invalid={!!formErrors.host}
                  />
                  {formErrors.host && <p className="text-sm text-destructive">{formErrors.host}</p>}
                </div>
                <div className="space-y-2 col-span-1">
                  <label htmlFor="create-port" className="text-sm font-medium">
                    Port *
                  </label>
                  <Input
                    id="create-port"
                    type="number"
                    value={formData.port}
                    onChange={(e) =>
                      setFormData({ ...formData, port: parseInt(e.target.value) || 0 })
                    }
                    placeholder="Enter database port"
                    min={1}
                    max={65535}
                    aria-invalid={!!formErrors.port}
                  />
                  {formErrors.port && <p className="text-sm text-destructive">{formErrors.port}</p>}
                </div>
              </div>
              <div className="space-y-2">
                <label htmlFor="create-username" className="text-sm font-medium">
                  Username *
                </label>
                <Input
                  id="create-username"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  placeholder="Enter database username"
                  aria-invalid={!!formErrors.username}
                />
                {formErrors.username && (
                  <p className="text-sm text-destructive">{formErrors.username}</p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="create-password" className="text-sm font-medium">
                  Password (optional)
                </label>
                <Input
                  id="create-password"
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder="Enter database password"
                  disabled={formData.use_rds_iam_auth}
                  aria-invalid={!!formErrors.password}
                />
                {formErrors.password && (
                  <p className="text-sm text-destructive">{formErrors.password}</p>
                )}
              </div>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.use_rds_iam_auth}
                    onChange={(e) =>
                      setFormData({ ...formData, use_rds_iam_auth: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-gray-300"
                  />
                  <span className="text-sm font-medium">Use RDS IAM Authentication</span>
                </label>
              </div>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.require_ssl}
                    onChange={(e) => setFormData({ ...formData, require_ssl: e.target.checked })}
                    className="h-4 w-4 rounded border-gray-300"
                  />
                  <span className="text-sm font-medium">Require SSL</span>
                </label>
              </div>

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
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <label htmlFor="edit-name" className="text-sm font-medium">
                  Instance Name *
                </label>
                <Input
                  id="edit-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Enter instance name"
                  aria-invalid={!!formErrors.name}
                />
                {formErrors.name && <p className="text-sm text-destructive">{formErrors.name}</p>}
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-description" className="text-sm font-medium">
                  Description
                </label>
                <textarea
                  id="edit-description"
                  value={formData.description || ''}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Enter instance description"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-type" className="text-sm font-medium">
                  Database Engine *
                </label>
                <select
                  id="edit-type"
                  value={formData.database_engine}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      database_engine: e.target.value as 'mysql' | 'postgres',
                    })
                  }
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="postgresql">PostgreSQL</option>
                  <option value="mysql">MySQL</option>
                </select>
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-environment" className="text-sm font-medium">
                  Environment (optional)
                </label>
                <select
                  id="edit-environment"
                  value={formData.environment_id || ''}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      environment_id: e.target.value || undefined,
                    })
                  }
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <option value="">No Environment</option>
                  {environments.map((env) => (
                    <option key={env.id} value={env.id}>
                      {env.name}
                    </option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label htmlFor="edit-host" className="text-sm font-medium">
                    Host *
                  </label>
                  <Input
                    id="edit-host"
                    value={formData.host}
                    onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                    placeholder="Enter database host"
                    aria-invalid={!!formErrors.host}
                  />
                  {formErrors.host && <p className="text-sm text-destructive">{formErrors.host}</p>}
                </div>
                <div className="space-y-2">
                  <label htmlFor="edit-port" className="text-sm font-medium">
                    Port *
                  </label>
                  <Input
                    id="edit-port"
                    type="number"
                    value={formData.port}
                    onChange={(e) =>
                      setFormData({ ...formData, port: parseInt(e.target.value) || 0 })
                    }
                    placeholder="Enter database port"
                    min={1}
                    max={65535}
                    aria-invalid={!!formErrors.port}
                  />
                  {formErrors.port && <p className="text-sm text-destructive">{formErrors.port}</p>}
                </div>
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-username" className="text-sm font-medium">
                  Username *
                </label>
                <Input
                  id="edit-username"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  placeholder="Enter database username"
                  aria-invalid={!!formErrors.username}
                />
                {formErrors.username && (
                  <p className="text-sm text-destructive">{formErrors.username}</p>
                )}
              </div>
              <div className="space-y-2">
                <label htmlFor="edit-password" className="text-sm font-medium">
                  Password (optional, leave blank to keep current)
                </label>
                <Input
                  id="edit-password"
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder="Enter new password (leave blank to keep current)"
                  disabled={formData.use_rds_iam_auth}
                  aria-invalid={!!formErrors.password}
                />
                {formErrors.password && (
                  <p className="text-sm text-destructive">{formErrors.password}</p>
                )}
              </div>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.use_rds_iam_auth}
                    onChange={(e) =>
                      setFormData({ ...formData, use_rds_iam_auth: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-gray-300"
                  />
                  <span className="text-sm font-medium">Use RDS IAM Authentication</span>
                </label>
              </div>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.require_ssl}
                    onChange={(e) => setFormData({ ...formData, require_ssl: e.target.checked })}
                    className="h-4 w-4 rounded border-gray-300"
                  />
                  <span className="text-sm font-medium">Require SSL</span>
                </label>
              </div>

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
