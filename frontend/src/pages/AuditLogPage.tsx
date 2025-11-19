import React, { useState, useMemo } from 'react';
import { useAuth } from '@/contexts/AuthContext';
import { DefaultLayout } from '@/components/layouts';
import type { ListAuditLogsParams, AuditLog } from '@/lib/api/types';
import { useUsers } from '@/hooks/api/useUsersSWR';
import { useAuditLogsSWR } from '@/hooks/api/useAuditLogsSWR';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { DataTable } from '@/components/data-table';
import { getAuditLogsColumns } from './audit-logs/columns';
import { AlertTriangle, Filter, X, Calendar, User, Activity, Database } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { CodeBlock } from '@/components/ui/code-block';

// Common actions and resource types for filters
// Actions match backend constants (lowercase with underscores)
const COMMON_ACTIONS = ['login', 'logout', 'register', 'create', 'read', 'update', 'delete'];

// Resource types match backend constants (lowercase with underscores)
const RESOURCE_TYPES = [
  'user',
  'project',
  'environment',
  'database',
  'database_instance',
  'change_plan',
  'approval',
  'execution_log',
  'changelog',
  'schema_history',
  'audit_log',
  'policy',
  'policy_finding',
];

export const AuditLogPage: React.FC = () => {
  const { user } = useAuth();

  // Pagination state
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState(50);

  // Use cached users from store instead of fetching
  const { getUserById } = useUsers();

  // Filters
  const [filters, setFilters] = useState<ListAuditLogsParams>({});

  // Sheet state for audit log details
  const [selectedAuditLog, setSelectedAuditLog] = useState<AuditLog | null>(null);
  const [isSheetOpen, setIsSheetOpen] = useState(false);

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  // Construct params for SWR hook
  const params = useMemo(
    () => ({
      ...filters,
      limit: pageSize,
      offset: pageIndex * pageSize,
    }),
    [filters, pageSize, pageIndex],
  );

  // Use SWR hook for data fetching
  const { auditLogs, total, isLoading, error } = useAuditLogsSWR(params, isAdmin);

  const handleFilterChange = (key: keyof ListAuditLogsParams, value: string | undefined) => {
    setFilters((prev) => {
      const newFilters = { ...prev };
      if (value === '' || value === undefined) {
        delete newFilters[key];
      } else {
        (newFilters as Record<string, string>)[key] = value;
      }
      return newFilters;
    });
    setPageIndex(0);
  };

  const handleDateFilterChange = (key: 'start_date' | 'end_date', value: string) => {
    setFilters((prev) => {
      const newFilters = { ...prev };
      if (value === '') {
        delete newFilters[key];
      } else {
        // Convert to RFC3339 format
        const date = new Date(value);
        if (!isNaN(date.getTime())) {
          newFilters[key] = date.toISOString();
        }
      }
      return newFilters;
    });
    setPageIndex(0);
  };

  const clearFilters = () => {
    setFilters({});
    setPageIndex(0);
  };

  // Handler for row clicks
  const handleRowClick = (auditLog: AuditLog) => {
    setSelectedAuditLog(auditLog);
    setIsSheetOpen(true);
  };

  // Get columns for audit logs table
  const auditLogsColumns = getAuditLogsColumns({
    getUserById,
  });

  if (!isAdmin) {
    return (
      <DefaultLayout breadcrumbs={[{ label: 'Audit Logs', isCurrentPage: true }]}>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <div className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-4 w-4" />
              <span className="text-sm font-medium">Access Denied</span>
            </div>
            <p className="text-sm text-muted-foreground mt-2">
              Admin access is required to view audit logs.
            </p>
          </CardContent>
        </Card>
      </DefaultLayout>
    );
  }

  return (
    <DefaultLayout breadcrumbs={[{ label: 'Audit Logs', isCurrentPage: true }]}>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Audit Logs</h1>
          <p className="text-muted-foreground mt-1">
            View and filter audit log entries for system activities
          </p>
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

        {/* Filters */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Filter className="h-4 w-4" />
                <CardTitle>Filters</CardTitle>
              </div>
              {(filters.user_id ||
                filters.action ||
                filters.target_type ||
                filters.target_id ||
                filters.start_date ||
                filters.end_date) && (
                <Button variant="ghost" size="sm" onClick={clearFilters}>
                  <X className="h-4 w-4 mr-1" />
                  Clear Filters
                </Button>
              )}
            </div>
            <CardDescription>Filter audit logs by various criteria</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {/* User Filter */}
              <div className="space-y-2">
                <Label htmlFor="user-filter">User</Label>
                <Input
                  id="user-filter"
                  placeholder="User ID"
                  value={filters.user_id || ''}
                  onChange={(e) => handleFilterChange('user_id', e.target.value || undefined)}
                />
              </div>

              {/* Action Filter */}
              <div className="space-y-2">
                <Label htmlFor="action-filter">Action</Label>
                <Select
                  value={filters.action || undefined}
                  onValueChange={(value) => handleFilterChange('action', value || undefined)}
                >
                  <SelectTrigger id="action-filter">
                    <SelectValue placeholder="All actions" />
                  </SelectTrigger>
                  <SelectContent>
                    {COMMON_ACTIONS.map((action) => (
                      <SelectItem key={action} value={action}>
                        {action}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Resource Type Filter */}
              <div className="space-y-2">
                <Label htmlFor="resource-type-filter">Resource Type</Label>
                <Select
                  value={filters.target_type || undefined}
                  onValueChange={(value) => handleFilterChange('target_type', value || undefined)}
                >
                  <SelectTrigger id="resource-type-filter">
                    <SelectValue placeholder="All resource types" />
                  </SelectTrigger>
                  <SelectContent>
                    {RESOURCE_TYPES.map((type) => (
                      <SelectItem key={type} value={type}>
                        {type}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Target ID Filter */}
              <div className="space-y-2">
                <Label htmlFor="target-id-filter">Target ID</Label>
                <Input
                  id="target-id-filter"
                  placeholder="Resource ID"
                  value={filters.target_id || ''}
                  onChange={(e) => handleFilterChange('target_id', e.target.value || undefined)}
                />
              </div>

              {/* Start Date Filter */}
              <div className="space-y-2">
                <Label htmlFor="start-date-filter">Start Date</Label>
                <Input
                  id="start-date-filter"
                  type="datetime-local"
                  value={
                    filters.start_date
                      ? new Date(filters.start_date).toISOString().slice(0, 16)
                      : ''
                  }
                  onChange={(e) => handleDateFilterChange('start_date', e.target.value)}
                />
              </div>

              {/* End Date Filter */}
              <div className="space-y-2">
                <Label htmlFor="end-date-filter">End Date</Label>
                <Input
                  id="end-date-filter"
                  type="datetime-local"
                  value={
                    filters.end_date ? new Date(filters.end_date).toISOString().slice(0, 16) : ''
                  }
                  onChange={(e) => handleDateFilterChange('end_date', e.target.value)}
                />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Audit Logs Table */}
        <Card>
          <CardHeader>
            <CardTitle>Audit Log Entries</CardTitle>
            <CardDescription>
              {isLoading ? (
                'Loading audit logs...'
              ) : (
                <>
                  {total > 0
                    ? `Showing ${auditLogs.length} of ${total} entries`
                    : 'No audit log entries found'}
                </>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={auditLogsColumns}
              data={auditLogs}
              isLoading={isLoading}
              onRowClick={handleRowClick}
              emptyMessage={
                filters.user_id ||
                filters.action ||
                filters.target_type ||
                filters.target_id ||
                filters.start_date ||
                filters.end_date
                  ? 'No audit log entries match your filters.'
                  : 'No audit log entries found.'
              }
              pagination={{
                pageIndex,
                pageSize,
                totalCount: total,
                onPageChange: setPageIndex,
                onPageSizeChange: (newSize) => {
                  setPageSize(newSize);
                  setPageIndex(0);
                },
                pageSizeOptions: [10, 20, 50, 100],
              }}
            />
          </CardContent>
        </Card>

        {/* Audit Log Detail Sheet */}
        <Sheet open={isSheetOpen} onOpenChange={setIsSheetOpen}>
          <SheetContent side="right" className="w-full sm:max-w-2xl overflow-y-auto">
            <SheetHeader>
              <SheetTitle>Audit Log Details</SheetTitle>
              <SheetDescription>Detailed information about this audit log entry</SheetDescription>
            </SheetHeader>

            {selectedAuditLog && (
              <Tabs defaultValue="overview" className="mt-6">
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value="overview">Overview</TabsTrigger>
                  <TabsTrigger value="info">Info</TabsTrigger>
                </TabsList>

                {/* Overview Tab */}
                <TabsContent value="overview" className="space-y-4 mt-4">
                  {/* Main Metadata Card */}
                  <Card>
                    <CardContent className="pt-4">
                      <div className="grid grid-cols-2 gap-x-4 gap-y-3 text-xs">
                        {/* Timestamp */}
                        <div className="flex items-start gap-2">
                          <Calendar className="h-3.5 w-3.5 text-muted-foreground mt-0.5 shrink-0" />
                          <div className="flex-1 min-w-0">
                            <p className="text-muted-foreground">Timestamp</p>
                            <p className="text-sm font-medium">
                              {new Date(selectedAuditLog.timestamp).toLocaleString('en-US', {
                                month: 'short',
                                day: 'numeric',
                                hour: '2-digit',
                                minute: '2-digit',
                              })}
                            </p>
                          </div>
                        </div>

                        {/* User */}
                        <div className="flex items-start gap-2">
                          <User className="h-3.5 w-3.5 text-muted-foreground mt-0.5 shrink-0" />
                          <div className="flex-1 min-w-0">
                            <p className="text-muted-foreground">User</p>
                            <p className="text-sm font-medium truncate">
                              {getUserById(selectedAuditLog.user_id)?.email || 'Unknown'}
                            </p>
                          </div>
                        </div>

                        {/* Action */}
                        <div className="flex items-start gap-2">
                          <Activity className="h-3.5 w-3.5 text-muted-foreground mt-0.5 shrink-0" />
                          <div className="flex-1 min-w-0">
                            <p className="text-muted-foreground">Action</p>
                            <Badge variant="outline" className="text-xs mt-0.5">
                              {selectedAuditLog.action}
                            </Badge>
                          </div>
                        </div>

                        {/* Resource Type */}
                        <div className="flex items-start gap-2">
                          <Database className="h-3.5 w-3.5 text-muted-foreground mt-0.5 shrink-0" />
                          <div className="flex-1 min-w-0">
                            <p className="text-muted-foreground">Resource Type</p>
                            <Badge variant="secondary" className="text-xs mt-0.5">
                              {selectedAuditLog.resource_type}
                            </Badge>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>

                  {/* Details - Full Space */}
                  {selectedAuditLog.details && Object.keys(selectedAuditLog.details).length > 0 && (
                    <div className="space-y-2">
                      <h3 className="text-sm font-semibold">Details</h3>
                      <CodeBlock
                        language="json"
                        filename="details.json"
                        code={JSON.stringify(selectedAuditLog.details, null, 2)}
                      />
                    </div>
                  )}
                </TabsContent>

                {/* Info Tab */}
                <TabsContent value="info" className="space-y-4 mt-4">
                  <Card>
                    <CardContent className="pt-4">
                      <div className="space-y-3 text-sm font-mono">
                        {/* Resource ID */}
                        <div>
                          <p className="text-xs text-muted-foreground mb-1">Resource ID:</p>
                          <code className="bg-muted px-2 py-1 rounded block break-all">
                            {selectedAuditLog.resource_id}
                          </code>
                        </div>

                        {/* IP Address */}
                        <div>
                          <p className="text-xs text-muted-foreground mb-1">IP:</p>
                          <code className="bg-muted px-2 py-1 rounded block">
                            {selectedAuditLog.ip_address || 'N/A'}
                          </code>
                        </div>

                        {/* User ID */}
                        <div>
                          <p className="text-xs text-muted-foreground mb-1">User ID:</p>
                          <code className="bg-muted px-2 py-1 rounded block break-all">
                            {selectedAuditLog.user_id?.toString() || 'N/A'}
                          </code>
                        </div>

                        {/* User Agent */}
                        {selectedAuditLog.user_agent && (
                          <div>
                            <p className="text-xs text-muted-foreground mb-1">UA:</p>
                            <code className="bg-muted px-2 py-1 rounded block break-all">
                              {selectedAuditLog.user_agent}
                            </code>
                          </div>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </TabsContent>
              </Tabs>
            )}
          </SheetContent>
        </Sheet>
      </div>
    </DefaultLayout>
  );
};
