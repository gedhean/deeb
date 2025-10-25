import React from 'react';
import { DefaultLayout } from '@/components/layouts';

export const DashboardPage: React.FC = () => {
  return (
    <DefaultLayout
      breadcrumbs={[
        {
          label: 'Building Your Application',
          href: '#',
        },
        {
          label: 'Data Fetching',
          isCurrentPage: true,
        },
      ]}
    >
      <div className="grid auto-rows-min gap-4 md:grid-cols-3">
        <div className="bg-muted/50 aspect-video rounded-xl" />
        <div className="bg-muted/50 aspect-video rounded-xl" />
        <div className="bg-muted/50 aspect-video rounded-xl" />
      </div>
      <div className="bg-muted/50 min-h-screen flex-1 rounded-xl md:min-h-min" />
    </DefaultLayout>
  );
};
