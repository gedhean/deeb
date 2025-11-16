import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { useProjectsSWR } from '@/hooks/api/useProjectsSWR';
import { useEnvironmentsSWR } from '@/hooks/api/useEnvironmentsSWR';
import { useDatabaseInstancesSWR } from '@/hooks/api/useDatabaseInstancesSWR';
import { useCreateDatabase } from '@/hooks/api/useDatabaseSWR';
import type { Database, ChangePlan } from '@/lib/api/types';

interface CreateDatabaseSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  databaseInstanceId?: string;
}

export const CreateDatabaseSheet: React.FC<CreateDatabaseSheetProps> = ({
  open,
  onOpenChange,
  databaseInstanceId,
}) => {
  const navigate = useNavigate();

  // Use SWR hooks for data fetching
  const { projects } = useProjectsSWR();
  const {
    environments,
    isLoading: environmentsLoading,
    error: environmentsError,
  } = useEnvironmentsSWR();
  const {
    databaseInstances: instances,
    isLoading: instancesLoading,
    error: instancesError,
  } = useDatabaseInstancesSWR();

  // Use mutation hook for creating database
  const { createDatabase, isLoading: loading } = useCreateDatabase();

  const [formData, setFormData] = useState({
    project_id: '',
    environment_id: '',
    database_instance_id: databaseInstanceId || '',
    database_name: '',
    description: '',
  });
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validate required fields
    if (
      !formData.project_id ||
      !formData.environment_id ||
      !formData.database_instance_id ||
      !formData.database_name
    ) {
      setError('Please fill in all required fields');
      return;
    }

    // Use mutation hook to create database
    const result = await createDatabase(formData.database_instance_id, {
      project_id: formData.project_id,
      environment_id: formData.environment_id,
      database_instance_id: formData.database_instance_id,
      database_name: formData.database_name,
      description: formData.description || undefined,
    });

    if (!result) {
      setError('Failed to create database');
      return;
    }

    // Check if result is Database or ChangePlan with more robust type checking
    if ('database_name' in result && 'database_instance_id' in result) {
      // It's a Database - redirect to database detail page
      const database = result as Database;
      navigate(`/databases/${database.id}`);
    } else if ('sql_content' in result && 'change_type' in result) {
      // It's a ChangePlan - redirect to change plan page
      const changePlan = result as ChangePlan;
      navigate(`/change-plans/${changePlan.id}`);
    } else {
      // Unexpected response format
      setError('Unexpected response format from server');
      return;
    }

    onOpenChange(false);
  };

  const handleClose = () => {
    if (!loading) {
      setFormData({
        project_id: '',
        environment_id: '',
        database_instance_id: databaseInstanceId || '',
        database_name: '',
        description: '',
      });
      setError(null);
      onOpenChange(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={handleClose}>
      <SheetContent className="sm:max-w-[525px] md:min-w-[600px] overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Create Database</SheetTitle>
          <SheetDescription>
            Create a new database or import an existing one from the database instance.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            {/* Project Field */}
            <div className="space-y-2">
              <label htmlFor="project" className="text-sm font-medium">
                Project *
              </label>
              <Select
                value={formData.project_id}
                onValueChange={(value) => setFormData({ ...formData, project_id: value })}
              >
                <SelectTrigger id="project">
                  <SelectValue placeholder="Select a project" />
                </SelectTrigger>
                <SelectContent>
                  {projects?.map((project) => (
                    <SelectItem key={project.id} value={project.id}>
                      {project.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Environment Field */}
            <div className="space-y-2">
              <label htmlFor="environment" className="text-sm font-medium">
                Environment *
              </label>
              <Select
                value={formData.environment_id}
                onValueChange={(value) => setFormData({ ...formData, environment_id: value })}
                disabled={environmentsLoading || !!environmentsError}
              >
                <SelectTrigger id="environment">
                  <SelectValue
                    placeholder={
                      environmentsLoading
                        ? 'Loading environments...'
                        : environmentsError
                          ? 'Failed to load environments'
                          : 'Select an environment'
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  {environments?.map((env) => (
                    <SelectItem key={env.id} value={env.id}>
                      {env.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {environmentsError && <p className="text-sm text-destructive">{environmentsError}</p>}
            </div>

            {/* Database Instance Field (only show if not pre-selected) */}
            {!databaseInstanceId && (
              <div className="space-y-2">
                <label htmlFor="database_instance" className="text-sm font-medium">
                  Database Instance *
                </label>
                <Select
                  value={formData.database_instance_id}
                  onValueChange={(value) =>
                    setFormData({ ...formData, database_instance_id: value })
                  }
                  disabled={instancesLoading || !!instancesError}
                >
                  <SelectTrigger id="database_instance">
                    <SelectValue
                      placeholder={
                        instancesLoading
                          ? 'Loading database instances...'
                          : instancesError
                            ? 'Failed to load database instances'
                            : 'Select a database instance'
                      }
                    />
                  </SelectTrigger>
                  <SelectContent>
                    {instances?.map((instance) => (
                      <SelectItem key={instance.id} value={instance.id}>
                        {instance.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {instancesError && <p className="text-sm text-destructive">{instancesError}</p>}
              </div>
            )}

            {/* Database Name Field */}
            <div className="space-y-2">
              <label htmlFor="database_name" className="text-sm font-medium">
                Database Name *
              </label>
              <Input
                id="database_name"
                type="text"
                value={formData.database_name}
                onChange={(e) => setFormData({ ...formData, database_name: e.target.value })}
                placeholder="my_database"
                disabled={loading}
              />
            </div>

            {/* Description Field */}
            <div className="space-y-2">
              <label htmlFor="description" className="text-sm font-medium">
                Description
              </label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder="Optional description"
                disabled={loading}
                rows={3}
              />
            </div>

            {error && (
              <div className="text-sm text-destructive bg-destructive/10 p-3 rounded-md">
                {error}
              </div>
            )}
          </div>

          <SheetFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? 'Creating...' : 'Create Database'}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
};

// Export as CreateDatabaseDialog for backward compatibility
export const CreateDatabaseDialog = CreateDatabaseSheet;
