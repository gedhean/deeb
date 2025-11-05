import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { changePlanService } from '@/lib/api/services/changePlans';
import { projectsService } from '@/lib/api/services/projects';
import { databasesService } from '@/lib/api/services/databases';
import { usersService } from '@/lib/api/services/users';
import type { ChangePlan, ChangePlanStatus, Project, Database, User } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { SQLReviewResults } from '@/components/SQLReviewResults';
import { ApprovalActions } from '@/components/ApprovalActions';
import { ApprovalHistory } from '@/components/ApprovalHistory';
import { ExecuteButton } from '@/components/ExecuteButton';
import { ExecutionResults } from '@/components/ExecutionResults';
import { SubmitForReview } from '@/components/SubmitForReview';
import {
  ArrowLeft,
  AlertTriangle,
  Calendar,
  User as UserIcon,
  Database as DatabaseIcon,
  FileCode,
} from 'lucide-react';

export const ChangePlanDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [changePlan, setChangePlan] = useState<ChangePlan | null>(null);
  const [project, setProject] = useState<Project | null>(null);
  const [database, setDatabase] = useState<Database | null>(null);
  const [author, setAuthor] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (id) {
      loadChangePlan(id);
    }
  }, [id]);

  const loadChangePlan = async (changePlanId: string) => {
    try {
      setIsLoading(true);
      setError('');

      // Load change plan
      const changePlanData = await changePlanService.getChangePlan(changePlanId);
      setChangePlan(changePlanData);

      // Load related data in parallel
      const [projectsRes, databasesRes, usersRes] = await Promise.all([
        projectsService.listProjects({ limit: 100 }),
        databasesService.listDatabases({ limit: 100 }),
        usersService.listUsers({ limit: 100 }),
      ]);

      // Find related entities
      const foundProject = projectsRes.items.find((p) => p.id === changePlanData.project_id);
      const foundDatabase = databasesRes.items.find(
        (d) => d.id === changePlanData.target_database_id,
      );
      const foundAuthor = usersRes.items.find((u) => u.id === changePlanData.author_id);

      setProject(foundProject || null);
      setDatabase(foundDatabase || null);
      setAuthor(foundAuthor || null);
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to load change plan. Please try again.');
      }
      console.error('Failed to load change plan:', err);
    } finally {
      setIsLoading(false);
    }
  };

  // Get status badge variant
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

  // Get status badge color class
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
            <Button variant="ghost" size="icon" onClick={() => navigate('/change-plans')}>
              <ArrowLeft className="h-5 w-5" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold tracking-tight">{changePlan.title}</h1>
              <p className="text-muted-foreground mt-1">
                {changePlan.description || 'No description provided'}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Badge
              variant={getStatusBadgeVariant(changePlan.status)}
              className={getStatusBadgeClass(changePlan.status)}
            >
              {changePlan.status}
            </Badge>
            <SubmitForReview changePlan={changePlan} onSuccess={() => loadChangePlan(id!)} />
            <ApprovalActions changePlan={changePlan} onSuccess={() => loadChangePlan(id!)} />
            <ExecuteButton changePlan={changePlan} onSuccess={() => loadChangePlan(id!)} />
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

              <div className="space-y-3">
                <div className="flex items-start gap-3">
                  <DatabaseIcon className="h-4 w-4 text-muted-foreground mt-1" />
                  <div>
                    <p className="text-sm font-medium">Database</p>
                    <p className="text-sm text-muted-foreground">
                      {database?.database_name || changePlan.target_database_id}
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <FileCode className="h-4 w-4 text-muted-foreground mt-1" />
                  <div>
                    <p className="text-sm font-medium">Project</p>
                    <p className="text-sm text-muted-foreground">
                      {project?.name || changePlan.project_id}
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <UserIcon className="h-4 w-4 text-muted-foreground mt-1" />
                  <div>
                    <p className="text-sm font-medium">Author</p>
                    <p className="text-sm text-muted-foreground">
                      {author?.email || changePlan.author_id}
                    </p>
                  </div>
                </div>

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
                      {changePlan.execution_duration_ms && (
                        <p className="text-xs text-muted-foreground mt-1">
                          Duration: {changePlan.execution_duration_ms}ms
                        </p>
                      )}
                      {changePlan.affected_rows !== undefined && (
                        <p className="text-xs text-muted-foreground mt-1">
                          Affected rows: {changePlan.affected_rows}
                        </p>
                      )}
                    </div>
                  </div>
                )}
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
          onRefresh={() => {
            if (id) {
              loadChangePlan(id);
            }
          }}
        />

        {/* Approval History */}
        <ApprovalHistory changePlanId={changePlan.id} />

        {/* Execution Results */}
        <ExecutionResults changePlan={changePlan} />

        {/* SQL Content */}
        <Card>
          <CardHeader>
            <CardTitle>SQL Content</CardTitle>
            <CardDescription>SQL statements to be executed</CardDescription>
          </CardHeader>
          <CardContent>
            <pre className="p-4 bg-muted rounded-md overflow-x-auto">
              <code className="text-sm font-mono">{changePlan.sql_content}</code>
            </pre>
          </CardContent>
        </Card>

        {/* Rollback SQL */}
        {changePlan.rollback_sql && (
          <Card>
            <CardHeader>
              <CardTitle>Rollback SQL</CardTitle>
              <CardDescription>SQL statements to rollback the changes</CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="p-4 bg-muted rounded-md overflow-x-auto">
                <code className="text-sm font-mono">{changePlan.rollback_sql}</code>
              </pre>
            </CardContent>
          </Card>
        )}
      </div>
    </DefaultLayout>
  );
};
