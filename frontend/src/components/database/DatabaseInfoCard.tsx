import React from 'react';
import type { Database, DatabaseInstance, Environment, Project } from '@/lib/api/types';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { DatabaseStatusBadge } from '@/components/DatabaseStatusBadge';
import { Calendar, FolderGit2, Globe, Server } from 'lucide-react';

interface DatabaseInfoCardProps {
  database: Database | null;
  project: Project | null;
  environment: Environment | null;
  instance: DatabaseInstance | null;
  isLoading: boolean;
  onNavigate: (path: string) => void;
  formatDate: (date: string) => string;
}

export const DatabaseInfoCard: React.FC<DatabaseInfoCardProps> = ({
  database,
  project,
  environment,
  instance,
  isLoading,
  onNavigate,
  formatDate,
}) => {
  if (isLoading) {
    return (
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
    );
  }

  if (!database) {
    return null;
  }

  return (
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
            <DatabaseStatusBadge status={database.status} variant="database" />
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Project</p>
            <div className="flex items-center gap-2">
              <FolderGit2 className="h-4 w-4 text-muted-foreground" />
              {database.project_id ? (
                <Button
                  variant="link"
                  className="h-auto p-0 text-sm"
                  onClick={() => onNavigate(`/projects/${database.project_id}`)}
                >
                  {project?.name || 'Unknown'}
                </Button>
              ) : (
                <span className="text-sm text-muted-foreground">Unassigned</span>
              )}
            </div>
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Environment</p>
            <div className="flex items-center gap-2">
              <Globe className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm">
                {database.environment_id ? environment?.name || 'Unknown' : 'Unassigned'}
              </span>
            </div>
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Database Instance</p>
            <div className="flex items-center gap-2">
              <Server className="h-4 w-4 text-muted-foreground" />
              <Button
                variant="link"
                className="h-auto p-0 text-sm"
                onClick={() => onNavigate(`/database-instances/${database.database_instance_id}`)}
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
  );
};
