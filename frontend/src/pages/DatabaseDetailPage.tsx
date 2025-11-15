import React, { useCallback, useEffect, useRef, useState } from 'react';
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
  User,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
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
import { schemaService } from '@/lib/api/services/schema';
import type { DatabaseSchemaMetadata } from '@/lib/api/types';

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
  const [selectedTableName, setSelectedTableName] = useState<string | null>(null);
  const [isTableDetailOpen, setIsTableDetailOpen] = useState(false);
  const [schemaMetadata, setSchemaMetadata] = useState<DatabaseSchemaMetadata | null>(null);
  const [isLoadingSchema, setIsLoadingSchema] = useState(false);
  const [isRefreshingSchema, setIsRefreshingSchema] = useState(false);
  const [isRefreshDisabled, setIsRefreshDisabled] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Check if user has databases.read permission
  const canReadDatabases = hasOrganizationPermission(
    user?.organization_role,
    OrganizationPermission.DatabasesRead,
  );

  const loadChangePlans = useCallback(async (databaseId: string) => {
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
  }, []);

  const loadDatabase = useCallback(
    async (databaseId: string) => {
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
    },
    [loadChangePlans],
  );

  useEffect(() => {
    if (id) {
      loadDatabase(id);
    }
  }, [id, loadDatabase]);

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
    const author = users.find((u) => u.id === authorId);
    return author?.email || authorId;
  };

  // Handle table click
  const handleTableClick = (tableName: string) => {
    setSelectedTableName(tableName);
    setIsTableDetailOpen(true);
  };

  // Load schema metadata for views
  const loadSchemaMetadata = useCallback(async (databaseId: string) => {
    try {
      setIsLoadingSchema(true);
      const data = await schemaService.getDatabaseSchema(databaseId);
      setSchemaMetadata(data);
    } catch (err) {
      // Clear schema metadata on error so UI shows empty state
      setSchemaMetadata(null);

      // Show user-facing error notification
      if (err instanceof APIError) {
        console.error('Failed to load schema metadata:', err.message);
        toast.error('Failed to load schema metadata', {
          description: err.message,
        });
      } else {
        console.error('Failed to load schema metadata:', err);
        toast.error('Failed to load schema metadata', {
          description: 'An unexpected error occurred. Please try again.',
        });
      }
    } finally {
      setIsLoadingSchema(false);
    }
  }, []);

  // Refresh schema metadata with debounce and toast notifications
  const handleRefreshSchema = async () => {
    if (!database?.id || isRefreshDisabled || isRefreshingSchema) return;

    // Clear any existing timer before starting a new one
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current);
      refreshTimerRef.current = null;
    }

    setIsRefreshDisabled(true);

    try {
      setIsRefreshingSchema(true);
      toast.info('Schema refresh started', {
        description: 'The refresh is running in the background and may take a few moments.',
      });

      const data = await schemaService.refreshDatabaseSchema(database.id);
      setSchemaMetadata(data);

      toast.success('Schema refreshed successfully', {
        description: 'The database schema metadata has been updated.',
      });
    } catch (err) {
      if (err instanceof APIError) {
        toast.error('Failed to refresh schema', {
          description: err.message,
        });
        console.error('Failed to refresh schema metadata:', err.message);
      } else {
        toast.error('Failed to refresh schema', {
          description: 'An unexpected error occurred. Please try again.',
        });
        console.error('Failed to refresh schema metadata:', err);
      }
    } finally {
      // Enable debounce - disable button and loading state for 5 seconds
      refreshTimerRef.current = setTimeout(() => {
        setIsRefreshingSchema(false);
        setIsRefreshDisabled(false);
        refreshTimerRef.current = null;
      }, 5000);
    }
  };

  useEffect(() => {
    if (database?.id) {
      loadSchemaMetadata(database.id);
    }
  }, [database?.id, loadSchemaMetadata]);

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
            </TabsList>
            {activeTab === 'overview' && (
              <Button
                variant="outline"
                size="lg"
                onClick={handleRefreshSchema}
                disabled={isLoadingSchema || isRefreshingSchema || isRefreshDisabled}
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshingSchema ? 'animate-spin' : ''}`} />
                {isRefreshingSchema ? 'Refreshing...' : 'Refresh Schema'}
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
        </Tabs>
      </div>
    </DefaultLayout>
  );
};
