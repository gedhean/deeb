import React from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import useSWR from 'swr';
import { useTheme } from '@/components/theme-provider';
import { DefaultLayout } from '@/components/layouts';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { MonacoSQLEditor } from '@/components/editors/MonacoSQLEditor';
import { changelogService } from '@/lib/api/services/changelog';
import { SNAPSHOT_STATUS } from '@/lib/api/types';
import {
  ArrowLeft,
  AlertTriangle,
  Calendar,
  FileCode,
  Database as DatabaseIcon,
} from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

export const ChangelogDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { theme } = useTheme();

  // Fetch changelog details with SWR
  const {
    data: changelogDetail,
    isLoading,
    error,
  } = useSWR(id ? `/api/v1/changelog/${id}` : null, () =>
    id ? changelogService.getChangelogDetails(id) : null,
  );

  // Format date helper
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

  // Get status badge variant and color class
  const getStatusBadge = (status: string) => {
    if (status === 'done') {
      return {
        variant: 'default' as const,
        colorClass: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
      };
    }
    if (status === 'failed') {
      return {
        variant: 'destructive' as const,
        colorClass: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
      };
    }
    return {
      variant: 'secondary' as const,
      colorClass: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200',
    };
  };

  if (isLoading) {
    return (
      <DefaultLayout
        breadcrumbs={[
          { label: 'Changelog', href: '/databases' },
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

  if (error || !changelogDetail) {
    return (
      <DefaultLayout
        breadcrumbs={[
          { label: 'Changelog', href: '/databases' },
          { label: 'Error', isCurrentPage: true },
        ]}
      >
        <div className="space-y-6">
          <Card className="border-destructive">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-destructive">
                <AlertTriangle className="h-5 w-5" />
                Error Loading Changelog
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">
                {error ? String(error) : 'Changelog not found.'}
              </p>
              <Button variant="outline" onClick={() => navigate(-1)}>
                <ArrowLeft className="mr-2 h-4 w-4" />
                Go Back
              </Button>
            </CardContent>
          </Card>
        </div>
      </DefaultLayout>
    );
  }

  const { changelog, change_plan, schema_history } = changelogDetail;

  return (
    <DefaultLayout
      breadcrumbs={[
        { label: 'Databases', href: '/databases' },
        {
          label: 'Database',
          href: `/databases/${changelog.database_id}`,
        },
        { label: 'Changelog', isCurrentPage: true },
      ]}
    >
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => navigate(`/databases/${changelog.database_id}`)}
              aria-label="Back to database"
            >
              <ArrowLeft className="h-5 w-5" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold tracking-tight">Changelog</h1>
              <p className="text-muted-foreground mt-1">
                {formatDistanceToNow(new Date(changelog.created_at), { addSuffix: true })}
              </p>
            </div>
          </div>
          <Badge
            variant={getStatusBadge(changelog.status).variant}
            className={`capitalize text-base ${getStatusBadge(changelog.status).colorClass}`}
          >
            {changelog.status}
          </Badge>
        </div>

        {/* Changelog Information */}
        <Card>
          <CardHeader>
            <CardTitle>Changelog Details</CardTitle>
            <CardDescription>Information about this database changelog entry</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Left Column */}
              <div className="space-y-3">
                <div className="flex items-start gap-3">
                  <FileCode className="h-4 w-4 text-muted-foreground mt-1" />
                  <div className="flex-1">
                    <p className="text-sm font-medium">Change Plan</p>
                    <Button
                      variant="link"
                      className="h-auto p-0 text-sm text-muted-foreground hover:text-foreground"
                      onClick={() => navigate(`/change-plans/${change_plan.id}`)}
                    >
                      {change_plan.title}
                    </Button>
                    {change_plan.description && (
                      <p className="text-xs text-muted-foreground mt-1">
                        {change_plan.description}
                      </p>
                    )}
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <DatabaseIcon className="h-4 w-4 text-muted-foreground mt-1" />
                  <div className="flex-1">
                    <p className="text-sm font-medium">Database</p>
                    <Button
                      variant="link"
                      className="h-auto p-0 text-sm text-muted-foreground hover:text-foreground"
                      onClick={() => navigate(`/databases/${changelog.database_id}`)}
                    >
                      View Database
                    </Button>
                  </div>
                </div>
              </div>

              {/* Right Column */}
              <div className="space-y-3">
                <div className="flex items-start gap-3">
                  <Calendar className="h-4 w-4 text-muted-foreground mt-1" />
                  <div>
                    <p className="text-sm font-medium">Created</p>
                    <p className="text-sm text-muted-foreground">
                      {formatDate(changelog.created_at)}
                    </p>
                  </div>
                </div>

                <div className="flex items-start gap-3">
                  <FileCode className="h-4 w-4 text-muted-foreground mt-1" />
                  <div>
                    <p className="text-sm font-medium">Changelog ID</p>
                    <p className="text-sm text-muted-foreground font-mono">{changelog.id}</p>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Schema History */}
        {schema_history ? (
          <Card>
            <CardHeader>
              <CardTitle>Schema Snapshot</CardTitle>
              <CardDescription>
                Database schema snapshot captured at the time of this change
              </CardDescription>
            </CardHeader>
            <CardContent>
              <MonacoSQLEditor
                value={schema_history.raw_dump}
                onChange={() => {}}
                readonly={true}
                height={600}
                theme={theme === 'dark' ? 'dark' : 'light'}
                dialect="mysql"
              />
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardHeader>
              <CardTitle>Schema Snapshot</CardTitle>
              <CardDescription>
                {changelog.metadata?.snapshot_status === SNAPSHOT_STATUS.FAILED
                  ? 'Schema snapshot capture failed'
                  : 'No schema snapshot available'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {changelog.metadata?.snapshot_status === SNAPSHOT_STATUS.FAILED ? (
                <Alert
                  variant="destructive"
                  className="bg-amber-50 dark:bg-amber-950 border-amber-200 dark:border-amber-800"
                >
                  <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
                  <AlertTitle className="text-amber-900 dark:text-amber-100">
                    Snapshot Capture Failed
                  </AlertTitle>
                  <AlertDescription className="text-amber-800 dark:text-amber-200">
                    {changelog.metadata?.snapshot_error ||
                      'Unknown error occurred during snapshot capture'}
                  </AlertDescription>
                </Alert>
              ) : (
                <div className="flex items-center justify-center py-8 text-muted-foreground">
                  <p className="text-sm">
                    {changelog.status === 'pending'
                      ? 'Schema snapshot will be captured after execution'
                      : 'No schema snapshot was captured for this entry'}
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </DefaultLayout>
  );
};
