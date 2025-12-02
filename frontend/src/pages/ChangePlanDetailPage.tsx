import React, { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTheme } from '@/components/theme-provider';
import { DefaultLayout } from '@/components/layouts';
import { useProjectPermissions } from '@/hooks/api/useProjectPermissions';
import type { ChangePlanStatus } from '@/lib/api/types';
import { useUsers } from '@/hooks/api/useUsersSWR';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useDatabases } from '@/hooks/api/useDatabases';
import { useChangePlanSWR, useUpdateChangePlanSWR } from '@/hooks/api/useChangePlansSWR';
import { useProjectMembersSWR } from '@/hooks/api/useProjectMembersSWR';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { MonacoSQLEditor } from '@/components/editors/MonacoSQLEditor';
import { SQLReviewResults } from '@/components/SQLReviewResults';
import { ApprovalHistory } from '@/components/ApprovalHistory';
import { ExecutionResults } from '@/components/ExecutionResults';
import { ChangePlanStatusBadge } from '@/components/ChangePlanStatusBadge';
import { ChangePlanActions } from '@/components/ChangePlanActions';
import { showSuccessToast, handleApiError } from '@/lib/errorHandling';
import {
  ArrowLeft,
  AlertTriangle,
  Calendar,
  User as UserIcon,
  Database as DatabaseIcon,
  FileCode,
  Edit,
  Save,
  X,
  Loader2,
  Server,
} from 'lucide-react';

export const ChangePlanDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { theme } = useTheme();
  const [isEditingSQL, setIsEditingSQL] = useState(false);
  const [editingSQLContent, setEditingSQLContent] = useState('');
  const [editError, setEditError] = useState('');

  // Use SWR hook for change plan - automatic caching and revalidation
  const { changePlan, isLoading, error: swrError, refetch } = useChangePlanSWR(id);

  // Use cached data from store
  const { getUserById } = useUsers();
  const { getProjectById } = useProjectsSWR();
  const { getDatabaseById } = useDatabases();

  // Get related entities from cache
  const project = useMemo(() => {
    return changePlan?.project_id ? getProjectById(changePlan.project_id) : null;
  }, [changePlan, getProjectById]);

  const database = useMemo(() => {
    return changePlan ? getDatabaseById(changePlan.target_database_id) : null;
  }, [changePlan, getDatabaseById]);

  const author = useMemo(() => {
    return changePlan ? getUserById(changePlan.author_id) : null;
  }, [changePlan, getUserById]);

  // Use cached hook for project members - automatic deduplication
  const { members: projectMembers } = useProjectMembersSWR(project?.id);

  // Get project permissions based on loaded project members
  const permissions = useProjectPermissions(projectMembers);

  // Hook for updating change plan
  const { updateChangePlan, isLoading: isUpdatingSQL } = useUpdateChangePlanSWR();

  const error = swrError || '';

  // Format date
  const formatDate = (dateString: string): string => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'long',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return dateString;
    }
  };

  // Check if SQL content can be edited (not terminal states)
  // Terminal states are: Completed, Closed, Cancelled
  // Failed is no longer terminal and can be edited
  const canEditSQLContent = (status: ChangePlanStatus): boolean => {
    const terminalStates: ChangePlanStatus[] = ['Completed', 'Closed', 'Cancelled'];
    return !terminalStates.includes(status);
  };

  // Handle toggling edit mode
  const handleToggleEdit = () => {
    if (isEditingSQL) {
      // Cancel editing - revert changes
      setEditingSQLContent(changePlan?.sql_content || '');
      setEditError('');
    } else {
      // Start editing
      setEditingSQLContent(changePlan?.sql_content || '');
      setEditError('');
    }
    setIsEditingSQL(!isEditingSQL);
  };

  // Handle saving SQL content
  const handleSaveSQLContent = async () => {
    if (!changePlan || !editingSQLContent.trim()) {
      setEditError('SQL content is required');
      return;
    }

    setEditError('');

    const result = await updateChangePlan(changePlan.id, {
      sql_content: editingSQLContent.trim(),
    });

    if (result) {
      setIsEditingSQL(false);
      showSuccessToast('SQL content updated successfully. SQL Review will run automatically.');

      // Refetch change plan to get updated status and SQL Review results
      await refetch();
    } else {
      const errorMsg = handleApiError(new Error('Failed to update SQL content'), {
        showToast: true,
        toastTitle: 'Failed to update SQL content',
      });
      setEditError(errorMsg);
    }
  };

  if (isLoading) {
    return (
      <DefaultLayout
        breadcrumbs={[
          { label: 'Change Plans', href: '/change-plans' },
          { label: 'Loading...', isCurrentPage: true },
        ]}
      >
        <div className="space-y-6">
          <div className="flex items-center gap-4">
            <Skeleton className="h-10 w-10" />
            <Skeleton className="h-9 w-64" />
          </div>
          <Card>
            <CardHeader>
              <Skeleton className="h-8 w-48" />
              <Skeleton className="h-4 w-32 mt-2" />
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
              </div>
            </CardContent>
          </Card>
        </div>
      </DefaultLayout>
    );
  }

  if (error || !changePlan) {
    return (
      <DefaultLayout
        breadcrumbs={[
          { label: 'Change Plans', href: '/change-plans' },
          { label: 'Error', isCurrentPage: true },
        ]}
      >
        <div className="space-y-6">
          <Card className="border-destructive">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-destructive">
                <AlertTriangle className="h-5 w-5" />
                Error Loading Change Plan
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">
                {error || 'Change plan not found.'}
              </p>
              <Button variant="outline" onClick={() => navigate('/change-plans')}>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Back to Change Plans
              </Button>
            </CardContent>
          </Card>
        </div>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout
      breadcrumbs={[
        { label: 'Change Plans', href: '/change-plans' },
        { label: changePlan.title, isCurrentPage: true },
      ]}
    >
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div>
              <h1 className="text-3xl font-bold tracking-tight">{changePlan.title}</h1>
              <p className="text-muted-foreground mt-1">
                {changePlan.description || 'No description provided'}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <ChangePlanStatusBadge status={changePlan.status} />
            <ChangePlanActions
              changePlan={changePlan}
              onSuccess={() => refetch()}
              canApprove={permissions.canApproveChanges}
              canReject={permissions.canRejectChanges}
              canExecute={permissions.canExecuteChanges}
            />
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

        {/* Change Plan Information */}
        <div className="space-y-6">
          {/* Basic Information */}
          <Card className="w-full">
            <CardHeader>
              <CardTitle>Basic Information</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-2">
                <Badge variant="outline">{changePlan.change_type}</Badge>
                <Badge variant="outline">{changePlan.risk_level}</Badge>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                {/* Column 1: Database, Database Instance */}
                <div className="space-y-3">
                  <div className="flex items-start gap-3">
                    <DatabaseIcon className="h-4 w-4 text-muted-foreground mt-1" />
                    <div className="flex-1">
                      <p className="text-sm font-medium">Database</p>
                      {database ? (
                        <Button
                          variant="link"
                          className="h-auto p-0 text-sm text-muted-foreground hover:text-foreground"
                          onClick={() => navigate(`/databases/${database.id}`)}
                        >
                          {database.database_name}
                          {database.environment && (
                            <span className="-ml-1">({database.environment.name})</span>
                          )}
                        </Button>
                      ) : (
                        <p className="text-sm text-muted-foreground">
                          {changePlan.target_database_id}
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="flex items-start gap-3">
                    <Server className="h-4 w-4 text-muted-foreground mt-1" />
                    <div className="flex-1">
                      <p className="text-sm font-medium">Database Instance</p>
                      {database?.database_instance ? (
                        <Button
                          variant="link"
                          className="h-auto p-0 text-sm text-muted-foreground hover:text-foreground"
                          onClick={() =>
                            navigate(`/database-instances/${database.database_instance?.id}`)
                          }
                        >
                          {database.database_instance.name}
                        </Button>
                      ) : (
                        <p className="text-sm text-muted-foreground">
                          {database?.database_instance_id || 'N/A'}
                        </p>
                      )}
                    </div>
                  </div>
                </div>

                {/* Column 2: Project, Author */}
                <div className="space-y-3">
                  <div className="flex items-start gap-3">
                    <FileCode className="h-4 w-4 text-muted-foreground mt-1" />
                    <div className="flex-1">
                      <p className="text-sm font-medium">Project</p>
                      {project ? (
                        <Button
                          variant="link"
                          className="h-auto p-0 text-sm text-muted-foreground hover:text-foreground"
                          onClick={() => navigate(`/projects/${project.id}`)}
                        >
                          {project.name}
                        </Button>
                      ) : (
                        <p className="text-sm text-muted-foreground">
                          {changePlan.project_id || 'Unassigned'}
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="flex items-start gap-3">
                    <UserIcon className="h-4 w-4 text-muted-foreground mt-1" />
                    <div className="flex-1">
                      <p className="text-sm font-medium">Author</p>
                      {author ? (
                        <Button
                          variant="link"
                          className="h-auto p-0 text-sm text-muted-foreground hover:text-foreground"
                          onClick={() => navigate(`/users/${author.id}`)}
                        >
                          {author.email}
                        </Button>
                      ) : (
                        <p className="text-sm text-muted-foreground">{changePlan.author_id}</p>
                      )}
                    </div>
                  </div>
                </div>

                {/* Column 3: Created, Last Updated */}
                <div className="space-y-3">
                  <div className="flex items-start gap-3">
                    <Calendar className="h-4 w-4 text-muted-foreground mt-1" />
                    <div>
                      <p className="text-sm font-medium">Created</p>
                      <p className="text-sm text-muted-foreground">
                        {formatDate(changePlan.created_at)}
                      </p>
                    </div>
                  </div>

                  {changePlan.updated_at !== changePlan.created_at && (
                    <div className="flex items-start gap-3">
                      <Calendar className="h-4 w-4 text-muted-foreground mt-1" />
                      <div>
                        <p className="text-sm font-medium">Last Updated</p>
                        <p className="text-sm text-muted-foreground">
                          {formatDate(changePlan.updated_at)}
                        </p>
                      </div>
                    </div>
                  )}
                </div>

                {/* Column 4: Scheduled For, Executed At */}
                <div className="space-y-3">
                  {changePlan.scheduled_for && (
                    <div className="flex items-start gap-3">
                      <Calendar className="h-4 w-4 text-muted-foreground mt-1" />
                      <div>
                        <p className="text-sm font-medium">Scheduled For</p>
                        <p className="text-sm text-muted-foreground">
                          {formatDate(changePlan.scheduled_for)}
                        </p>
                      </div>
                    </div>
                  )}

                  {changePlan.executed_at && (
                    <div className="flex items-start gap-3">
                      <Calendar className="h-4 w-4 text-muted-foreground mt-1" />
                      <div>
                        <p className="text-sm font-medium">Executed At</p>
                        <p className="text-sm text-muted-foreground">
                          {formatDate(changePlan.executed_at)}
                        </p>
                        <div className="flex items-center gap-2">
                          {changePlan.execution_duration_ms && (
                            <p className="text-xs text-muted-foreground mt-1">
                              Duration: {changePlan.execution_duration_ms}ms
                            </p>
                          )}
                          {<span className="text-muted-foreground">&bull;</span>}
                          {changePlan.affected_rows !== undefined && (
                            <p className="text-xs text-muted-foreground mt-1">
                              Affected rows: {changePlan.affected_rows}
                            </p>
                          )}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Execution Status */}
          {changePlan.status === 'Failed' && changePlan.error_message && (
            <Card className="border-destructive w-full">
              <CardHeader>
                <CardTitle className="text-destructive">Execution Error</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground whitespace-pre-wrap">
                  {changePlan.error_message}
                </p>
              </CardContent>
            </Card>
          )}
        </div>

        {/* SQL Content */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <CardTitle>SQL Content</CardTitle>
                  {changePlan.statement_count !== undefined && changePlan.statement_count > 0 && (
                    <Badge variant="secondary" className="text-xs">
                      {changePlan.statement_count} statement
                      {changePlan.statement_count !== 1 ? 's' : ''}
                    </Badge>
                  )}
                </div>
                <CardDescription>SQL statements to be executed</CardDescription>
              </div>
              {permissions.canUpdateChanges && canEditSQLContent(changePlan.status) && (
                <div className="flex items-center gap-2">
                  {isEditingSQL ? (
                    <>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={handleToggleEdit}
                        disabled={isUpdatingSQL}
                        className="h-8"
                      >
                        <X className="mr-1 h-4 w-4" />
                        Cancel
                      </Button>
                      <Button
                        variant="default"
                        size="sm"
                        onClick={handleSaveSQLContent}
                        disabled={isUpdatingSQL}
                        className="h-8"
                      >
                        {isUpdatingSQL ? (
                          <>
                            <Loader2 className="mr-1 h-4 w-4 animate-spin" />
                            Saving...
                          </>
                        ) : (
                          <>
                            <Save className="mr-1 h-4 w-4" />
                            Save
                          </>
                        )}
                      </Button>
                    </>
                  ) : (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleToggleEdit}
                      className="h-8 w-8 p-0"
                      aria-label="Edit SQL content"
                    >
                      <Edit className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {isEditingSQL ? (
              <div className="space-y-2">
                <MonacoSQLEditor
                  value={editingSQLContent}
                  onChange={(value) => setEditingSQLContent(value)}
                  height={300}
                  theme={theme === 'dark' ? 'dark' : 'light'}
                  dialect={
                    database?.database_instance?.database_engine === 'postgres' ? 'pgsql' : 'mysql'
                  }
                />
                {editError && <p className="text-sm text-destructive">{editError}</p>}
                {changePlan.status !== 'Open' && (
                  <div className="rounded-md border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-950/20">
                    <div className="flex items-center gap-2 text-blue-800 dark:text-blue-200">
                      <AlertTriangle className="h-4 w-4" />
                      <span className="text-sm">
                        After saving, this change plan will transition to &quot;Open&quot; status
                        and SQL Review will run automatically.
                      </span>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <MonacoSQLEditor
                key={`readonly-${changePlan.updated_at}`}
                value={changePlan.sql_content}
                onChange={() => {}}
                readonly={true}
                height={300}
                theme={theme === 'dark' ? 'dark' : 'light'}
                dialect={
                  database?.database_instance?.database_engine === 'postgres' ? 'pgsql' : 'mysql'
                }
              />
            )}
          </CardContent>
        </Card>

        {/* Error blocking indication */}
        {changePlan.has_errors && changePlan.policy_findings && (
          <Card className="border-destructive">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-destructive">
                <AlertTriangle className="h-5 w-5" />
                SQL Review Errors Block Progression
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                This change plan has SQL Review errors that must be resolved before it can be
                approved or executed. Please review the errors below and update the SQL content.
              </p>
            </CardContent>
          </Card>
        )}

        {/* SQL Review Results */}
        <SQLReviewResults
          changePlanId={changePlan.id}
          findings={changePlan.policy_findings}
          changePlanStatus={changePlan.status}
          onRefresh={() => refetch()}
        />

        {/* Approval History */}
        <ApprovalHistory changePlanId={changePlan.id} />

        {/* Execution Results */}
        <ExecutionResults changePlan={changePlan} />

        {/* Rollback SQL */}
        {changePlan.rollback_sql && (
          <Card>
            <CardHeader>
              <CardTitle>Rollback SQL</CardTitle>
              <CardDescription>SQL statements to rollback the changes</CardDescription>
            </CardHeader>
            <CardContent>
              <MonacoSQLEditor
                value={changePlan.rollback_sql}
                onChange={() => {}}
                readonly={true}
                height={300}
                theme={theme === 'dark' ? 'dark' : 'light'}
                dialect={
                  database?.database_instance?.database_engine === 'postgres' ? 'pgsql' : 'mysql'
                }
              />
            </CardContent>
          </Card>
        )}
      </div>
    </DefaultLayout>
  );
};
