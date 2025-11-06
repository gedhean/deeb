import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { databasesService } from '@/lib/api/services/databases';
import { changePlanService } from '@/lib/api/services/changePlans';
import { projectsService } from '@/lib/api/services/projects';
import { environmentsService } from '@/lib/api/services/environments';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import { usersService } from '@/lib/api/services/users';
import type {
  Database,
  ChangePlan,
  Project,
  Environment,
  DatabaseInstance,
  ChangePlanStatus,
  User,
} from '@/lib/api/types';
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
  ExternalLink,
  FileEdit,
  Server,
  FolderGit2,
  Globe,
} from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';

export const DatabaseDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { user } = useAuth();
  const [database, setDatabase] = useState<Database | null>(null);
  const [changePlans, setChangePlans] = useState<ChangePlan[]>([]);
  const [project, setProject] = useState<Project | null>(null);
  const [environment, setEnvironment] = useState<Environment | null>(null);
  const [instance, setInstance] = useState<DatabaseInstance | null>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingChangePlans, setIsLoadingChangePlans] = useState(false);
  const [error, setError] = useState('');

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  useEffect(() => {
    if (id) {
      loadDatabase(id);
    }
  }, [id]);

  const loadDatabase = async (databaseId: string) => {
    try {
      setIsLoading(true);
      setError('');

      // Load database
      const databaseData = await databasesService.getDatabase(databaseId);
      setDatabase(databaseData);

      // Load change plans for this database
      await loadChangePlans(databaseId);

      // Load related data for display
      const [projectsRes, environmentsRes, instancesRes, usersRes] = await Promise.all([
        projectsService.listProjects({ limit: 100 }),
        environmentsService.listEnvironments({ limit: 100 }),
        databaseInstancesService.listDatabaseInstances({ limit: 100 }),
        usersService.listUsers({ limit: 100 }),
      ]);

      // Find related entities
      const foundProject = projectsRes.items.find((p) => p.id === databaseData.project_id);
      const foundEnvironment = environmentsRes.items.find(
        (e) => e.id === databaseData.environment_id,
      );
      const foundInstance = instancesRes.items.find(
        (i) => i.id === databaseData.database_instance_id,
      );

      setProject(foundProject || null);
      setEnvironment(foundEnvironment || null);
      setInstance(foundInstance || null);
      setUsers(usersRes.items);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load database. Please try again.');
      }
      console.error('Failed to load database:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const loadChangePlans = async (databaseId: string) => {
    try {
      setIsLoadingChangePlans(true);
      const response = await changePlanService.listChangePlans({
        database_id: databaseId,
        limit: 100,
      });
      setChangePlans(response.change_plans);
    } catch (err) {
      console.error('Failed to load change plans:', err);
    } finally {
      setIsLoadingChangePlans(false);
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
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
    }
  };

  // Get status badge variant for change plans
  const getStatusBadgeVariant = (
    status: ChangePlanStatus,
  ): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch (status) {
      case 'Open':
        return 'default';
      case 'Approved':
        return 'default';
      case 'Completed':
        return 'default';
      case 'Draft':
        return 'secondary';
      case 'InProgress':
        return 'outline';
      case 'Failed':
        return 'destructive';
      case 'Cancelled':
        return 'secondary';
      case 'Closed':
        return 'secondary';
      default:
        return 'secondary';
    }
  };

  // Get status badge color class for change plans
  const getStatusBadgeClass = (status: ChangePlanStatus): string => {
    switch (status) {
      case 'Open':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      case 'Approved':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'Completed':
        return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200';
      case 'Draft':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      case 'InProgress':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'Failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'Cancelled':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      case 'Closed':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
    }
  };

  // Get author name
  const getAuthorName = (authorId: string): string => {
    const author = users.find((u) => u.id === authorId);
    return author?.email || authorId;
  };

  // Render access denied
  if (!isAdmin) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Databases', isCurrentPage: true }]}>
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
              <p className="text-muted-foreground mt-1">
                {isLoading ? (
                  <Skeleton className="h-4 w-96 mt-2" />
                ) : (
                  database?.description || 'Database details'
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

        {/* Database Info */}
        {isLoading ? (
          <Card>
            <CardHeader>
              <CardTitle>Database Information</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            </CardContent>
          </Card>
        ) : database ? (
          <Card>
            <CardHeader>
              <CardTitle>Database Information</CardTitle>
              <CardDescription>Details about this database</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Database Name</p>
                  <p className="text-sm font-mono">{database.database_name}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Status</p>
                  <Badge className={getStatusColor(database.status)}>{database.status}</Badge>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Project</p>
                  <div className="flex items-center gap-2">
                    <FolderGit2 className="h-4 w-4 text-muted-foreground" />
                    <Button
                      variant="link"
                      className="h-auto p-0 text-sm"
                      onClick={() => navigate(`/projects/${database.project_id}`)}
                    >
                      {project?.name || database.project_id}
                    </Button>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Environment</p>
                  <div className="flex items-center gap-2">
                    <Globe className="h-4 w-4 text-muted-foreground" />
                    <span className="text-sm">{environment?.name || database.environment_id}</span>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Database Instance</p>
                  <div className="flex items-center gap-2">
                    <Server className="h-4 w-4 text-muted-foreground" />
                    <Button
                      variant="link"
                      className="h-auto p-0 text-sm"
                      onClick={() =>
                        navigate(`/database-instances/${database.database_instance_id}`)
                      }
                    >
                      {instance?.name || database.database_instance_id}
                    </Button>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">Created</p>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Calendar className="h-4 w-4" />
                    <span>{formatDate(database.created_at)}</span>
                  </div>
                </div>
                {database.description && (
                  <div className="space-y-1 md:col-span-2">
                    <p className="text-sm font-medium text-muted-foreground">Description</p>
                    <p className="text-sm">{database.description}</p>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        ) : null}

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
                  <Button onClick={() => navigate('/change-plans/create')}>
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
                          <Badge
                            variant={getStatusBadgeVariant(changePlan.status)}
                            className={getStatusBadgeClass(changePlan.status)}
                          >
                            {changePlan.status}
                          </Badge>
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
      </div>
    </DefaultLayout>
  );
};
