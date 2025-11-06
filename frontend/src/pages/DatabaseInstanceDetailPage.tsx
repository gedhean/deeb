import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import { databasesService } from '@/lib/api/services/databases';
import { projectsService } from '@/lib/api/services/projects';
import { environmentsService } from '@/lib/api/services/environments';
import type { DatabaseInstance, Database, Project, Environment } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
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
import {
  ArrowLeft,
  AlertTriangle,
  Calendar,
  Server,
  Database as DatabaseIcon,
  ExternalLink,
} from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';

export const DatabaseInstanceDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const [instance, setInstance] = useState<DatabaseInstance | null>(null);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingDatabases, setIsLoadingDatabases] = useState(false);
  const [error, setError] = useState('');

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  useEffect(() => {
    if (id) {
      loadInstance(id);
    }
  }, [id]);

  const loadInstance = async (instanceId: string) => {
    try {
      setIsLoading(true);
      setError('');

      // Load instance
      const instanceData = await databaseInstancesService.getDatabaseInstance(instanceId);
      setInstance(instanceData);

      // Load databases for this instance
      await loadDatabases(instanceId);

      // Load related data for display
      const [projectsRes, environmentsRes] = await Promise.all([
        projectsService.listProjects({ limit: 100 }),
        environmentsService.listEnvironments({ limit: 100 }),
      ]);

      setProjects(projectsRes.items);
      setEnvironments(environmentsRes.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load database instance. Please try again.');
      }
      console.error('Failed to load database instance:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const loadDatabases = async (instanceId: string) => {
    try {
      setIsLoadingDatabases(true);
      const response = await databasesService.listDatabases({
        database_instance_id: instanceId,
        limit: 100,
      });
      setDatabases(response.items);
    } catch (err) {
      console.error('Failed to load databases:', err);
    } finally {
      setIsLoadingDatabases(false);
    }
  };

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

  // Get status badge color
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Active':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'Inactive':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      case 'Maintenance':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
    }
  };

  // Get project name
  const getProjectName = (projectId: string): string => {
    const project = projects.find((p) => p.id === projectId);
    return project?.name || 'Unknown';
  };

  // Get environment name
  const getEnvironmentName = (environmentId: string): string => {
    const env = environments.find((e) => e.id === environmentId);
    return env?.name || 'Unknown';
  };

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
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Name</p>
                  <p className="text-sm">{instance.name}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Status</p>
                  <Badge className={getStatusColor(instance.status)}>{instance.status}</Badge>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Database Type</p>
                  <p className="text-sm font-mono">{instance.database_type}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Host</p>
                  <p className="text-sm font-mono">
                    {instance.host}:{instance.port}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Username</p>
                  <p className="text-sm font-mono">{instance.username}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Authentication</p>
                  <p className="text-sm">
                    {instance.use_rds_iam_auth ? 'RDS IAM Authentication' : 'Password'}
                  </p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">SSL Required</p>
                  <p className="text-sm">{instance.require_ssl ? 'Yes' : 'No'}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Created</p>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Calendar className="h-4 w-4" />
                    <span>{formatDate(instance.created_at)}</span>
                  </div>
                </div>
                {instance.description && (
                  <div className="space-y-1 md:col-span-2">
                    <p className="text-sm font-medium text-muted-foreground">Description</p>
                    <p className="text-sm">{instance.description}</p>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        ) : null}

        {/* Databases Table */}
        <Card>
          <CardHeader>
            <CardTitle>Databases</CardTitle>
            <CardDescription>
              {isLoadingDatabases ? (
                'Loading databases...'
              ) : (
                <>
                  {databases.length} database{databases.length === 1 ? '' : 's'} using this instance
                </>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {isLoadingDatabases ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Database Name</TableHead>
                    <TableHead>Project</TableHead>
                    <TableHead>Environment</TableHead>
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
            ) : databases.length === 0 ? (
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
              <div className="overflow-x-auto -mx-6 px-6">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Database Name</TableHead>
                      <TableHead>Project</TableHead>
                      <TableHead>Environment</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {databases.map((database) => (
                      <TableRow key={database.id}>
                        <TableCell className="font-medium">{database.database_name}</TableCell>
                        <TableCell className="text-muted-foreground">
                          {getProjectName(database.project_id)}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {getEnvironmentName(database.environment_id)}
                        </TableCell>
                        <TableCell>
                          <Badge variant={database.status === 'Active' ? 'default' : 'secondary'}>
                            {database.status}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(database.created_at)}
                        </TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => navigate(`/databases`)}
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
      </div>
    </DefaultLayout>
  );
};
