import React, { useEffect, useMemo } from 'react';
import { DefaultLayout } from '@/components/layouts';
import { useCurrentProject, useSetCurrentProject } from '@/store';

export const DashboardPage: React.FC = () => {
  const currentProject = useCurrentProject();
  const setCurrentProject = useSetCurrentProject();

  // Mock projects data - replace with actual API call
  const projects = useMemo(
    () => [
      {
        id: '1',
        name: 'Project Alpha',
        description: 'Alpha project',
        created_at: '',
        updated_at: '',
      },
      {
        id: '2',
        name: 'Project Beta',
        description: 'Beta project',
        created_at: '',
        updated_at: '',
      },
    ],
    [],
  );

  // Set default project if none is selected
  useEffect(() => {
    if (!currentProject && projects.length > 0) {
      setCurrentProject(projects[0]);
    }
  }, [currentProject, setCurrentProject, projects]);

  return (
    <DefaultLayout projects={projects} breadcrumbs={[]}>
      <div className="grid auto-rows-min gap-4 md:grid-cols-3">
        <div className="bg-muted/50 aspect-video rounded-xl" />
        <div className="bg-muted/50 aspect-video rounded-xl" />
        <div className="bg-muted/50 aspect-video rounded-xl" />
      </div>
      <div className="bg-muted/50 min-h-screen flex-1 rounded-xl md:min-h-min" />
    </DefaultLayout>
  );
};
