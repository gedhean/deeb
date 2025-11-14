import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { DefaultLayout } from '@/components/layouts';
import { changePlanService } from '@/lib/api/services/changePlans';
import type { ChangePlan } from '@/lib/api/types';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { healthService } from '@/lib/api/services/health';
import { ChangePlanStatusBadge } from '@/components/ChangePlanStatusBadge';

export const DashboardPage: React.FC = () => {
  const [changePlans, setChangePlans] = useState<ChangePlan[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [systemStatus, setSystemStatus] = useState<'healthy' | 'unhealthy' | 'loading'>('loading');

  // Fetch change plans and system status
  useEffect(() => {
    const loadData = async () => {
      try {
        setIsLoading(true);

        // Fetch change plans (recent and open)
        const allChangePlans = await changePlanService.listChangePlans({ limit: 10 });

        setChangePlans(allChangePlans.change_plans);

        // Check system health
        try {
          await healthService.check();
          setSystemStatus('healthy');
        } catch {
          setSystemStatus('unhealthy');
        }
      } catch (error) {
        console.error('Failed to load dashboard data:', error);
        setSystemStatus('unhealthy');
      } finally {
        setIsLoading(false);
      }
    };

    loadData();
  }, []);

  // Calculate statistics
  const stats = useMemo(() => {
    const openCount = changePlans.filter((cp) => cp.status === 'Open').length;
    const pendingApprovals = changePlans.filter((cp) => cp.status === 'Open');
    const recentChangePlans = changePlans.slice(0, 5);

    return {
      openCount,
      pendingApprovalsCount: pendingApprovals.length,
      pendingApprovals,
      recentChangePlans,
    };
  }, [changePlans]);

  // Format date
  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      });
    } catch {
      return dateString;
    }
  };

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Dashboard', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Overview Cards */}
        <div className="grid gap-4 md:grid-cols-3">
          {isLoading ? (
            <>
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
            </>
          ) : (
            <>
              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Open Change Plans</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{stats.openCount}</div>
                  <p className="text-xs text-muted-foreground">Change plans awaiting action</p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">Pending Approvals</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{stats.pendingApprovalsCount}</div>
                  <p className="text-xs text-muted-foreground">Awaiting your approval</p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">System Status</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-2">
                    <div
                      className={`h-2 w-2 rounded-full ${
                        systemStatus === 'healthy'
                          ? 'bg-green-500'
                          : systemStatus === 'unhealthy'
                            ? 'bg-red-500'
                            : 'bg-yellow-500'
                      }`}
                    />
                    <span className="text-sm font-medium capitalize">{systemStatus}</span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">API and database status</p>
                </CardContent>
              </Card>
            </>
          )}
        </div>

        {/* Recent Change Plans and Pending Approvals */}
        <div className="grid gap-6 md:grid-cols-2">
          {/* Recent Change Plans */}
          <Card>
            <CardHeader>
              <CardTitle>Recent Change Plans</CardTitle>
              <CardDescription>Latest change plans across all projects</CardDescription>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-16" />
                  <Skeleton className="h-16" />
                  <Skeleton className="h-16" />
                </div>
              ) : stats.recentChangePlans.length === 0 ? (
                <p className="text-sm text-muted-foreground">No change plans found</p>
              ) : (
                <div className="space-y-3">
                  {stats.recentChangePlans.map((changePlan) => (
                    <Link
                      key={changePlan.id}
                      to={`/change-plans/${changePlan.id}`}
                      className="flex items-center justify-between border-b pb-3 last:border-0 hover:bg-muted/50 -mx-4 px-4 py-2 rounded-md transition-colors cursor-pointer"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <h4 className="text-sm font-medium truncate">{changePlan.title}</h4>
                          <ChangePlanStatusBadge status={changePlan.status} />
                        </div>
                        <p className="text-xs text-muted-foreground mt-1 truncate">
                          {changePlan.description || 'No description'}
                        </p>
                        <p className="text-xs text-muted-foreground mt-1">
                          {formatDate(changePlan.created_at)}
                        </p>
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Pending Approvals */}
          <Card>
            <CardHeader>
              <CardTitle>Pending Approvals</CardTitle>
              <CardDescription>Change plans waiting for your approval</CardDescription>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-16" />
                  <Skeleton className="h-16" />
                  <Skeleton className="h-16" />
                </div>
              ) : stats.pendingApprovals.length === 0 ? (
                <p className="text-sm text-muted-foreground">No pending approvals</p>
              ) : (
                <div className="space-y-3">
                  {stats.pendingApprovals.map((changePlan) => (
                    <Link
                      key={changePlan.id}
                      to={`/change-plans/${changePlan.id}`}
                      className="flex items-center justify-between border-b pb-3 last:border-0 hover:bg-muted/50 -mx-4 px-4 py-2 rounded-md transition-colors cursor-pointer"
                    >
                      <div className="flex-1 min-w-0">
                        <h4 className="text-sm font-medium truncate">{changePlan.title}</h4>
                        <p className="text-xs text-muted-foreground mt-1 truncate">
                          {changePlan.description || 'No description'}
                        </p>
                        <div className="flex items-center gap-2 mt-2">
                          <span className="text-xs text-muted-foreground">
                            {changePlan.risk_level} Risk
                          </span>
                          {changePlan.has_errors && (
                            <span className="text-xs text-red-600 dark:text-red-400">
                              Has Errors
                            </span>
                          )}
                          {changePlan.has_warnings && (
                            <span className="text-xs text-yellow-600 dark:text-yellow-400">
                              Has Warnings
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="ml-4 shrink-0">
                        <span className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background border border-input hover:bg-accent hover:text-accent-foreground h-8 px-3 py-1">
                          Review
                        </span>
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </DefaultLayout>
  );
};
