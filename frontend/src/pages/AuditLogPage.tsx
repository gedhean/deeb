import React, { useEffect, useState } from 'react';
import { useAuth } from '@/contexts/AuthContext';
import { DefaultLayout } from '@/components/layouts';
import { auditLogService } from '@/lib/api/services/auditLogs';
import { usersService } from '@/lib/api/services/users';
import type { AuditLog, ListAuditLogsParams, User } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { AlertTriangle, Calendar, Filter, X } from 'lucide-react';

// Common actions and resource types for filters
const COMMON_ACTIONS = [
  'CreateProject',
  'UpdateProject',
  'DeleteProject',
  'CreateEnvironment',
  'UpdateEnvironment',
  'DeleteEnvironment',
  'CreatePolicy',
  'UpdatePolicy',
  'DeletePolicy',
  'CreateDatabaseInstance',
  'UpdateDatabaseInstance',
  'DeleteDatabaseInstance',
  'CreateDatabase',
  'UpdateDatabase',
  'DeleteDatabase',
  'CreateChangePlan',
  'UpdateChangePlan',
  'ApproveChangePlan',
  'CloseChangePlan',
  'ExecuteChangePlan',
  'CreateUser',
  'UpdateUser',
  'DeleteUser',
];

const RESOURCE_TYPES = [
  'Project',
  'Environment',
  'Policy',
  'DatabaseInstance',
  'Database',
  'ChangePlan',
  'User',
];

export const AuditLogPage: React.FC = () => {
  const { user } = useAuth();
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [users, setUsers] = useState<Map<string, User>>(new Map());
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [limit] = useState(50);

  // Filters
  const [filters, setFilters] = useState<ListAuditLogsParams>({
    limit,
    offset: 0,
  });

  // Check if user is admin
  const isAdmin = user?.organization_role === 'OrganizationAdmin';

  useEffect(() => {
    if (!isAdmin) {
      setError('Access denied. Admin access required.');
      setIsLoading(false);
      return;
    }

    loadAuditLogs();
    loadUsers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAdmin, filters, currentPage]);

  const loadUsers = async () => {
    try {
      const response = await usersService.listUsers({ limit: 100 });
      const usersMap = new Map<string, User>();
      response.items.forEach((u) => {
        usersMap.set(u.id, u);
      });
      setUsers(usersMap);
    } catch (err) {
      console.error('Failed to load users:', err);
    }
  };

  const loadAuditLogs = async () => {
    try {
      setIsLoading(true);
      setError('');

      const params: ListAuditLogsParams = {
        ...filters,
        limit,
        offset: (currentPage - 1) * limit,
      };

      const response = await auditLogService.listAuditLogs(params);
      setAuditLogs(response.audit_logs || []);
      setTotal(response.total || 0);
    } catch (err) {
      if (err instanceof APIError) {
        if (err.statusCode === 403) {
          setError('Access denied. Admin access required.');
        } else {
          setError(err.message || 'Failed to load audit logs.');
        }
      } else {
        setError('Failed to load audit logs. Please try again.');
      }
      console.error('Failed to load audit logs:', err);
    } finally {
      setIsLoading(false);
    }
  };

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
    setCurrentPage(1);
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
    setCurrentPage(1);
  };

  const clearFilters = () => {
    setFilters({ limit, offset: 0 });
    setCurrentPage(1);
  };

  const formatDate = (dateString: string): string => {
    try {
      return new Date(dateString).toLocaleString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return dateString;
    }
  };

  const formatDetails = (details?: Record<string, unknown>): string => {
    if (!details) return '-';
    try {
      return JSON.stringify(details, null, 2);
    } catch {
      return String(details);
    }
  };

  const totalPages = Math.ceil(total / limit);

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
                  value={filters.action || ""}
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
              {total > 0 ? (
                <>
                  Showing {auditLogs.length} of {total} entries
                </>
              ) : (
                'No audit log entries found'
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-4">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : auditLogs.length === 0 ? (
              <div className="text-center py-12">
                <p className="text-muted-foreground">No audit log entries found.</p>
                {(filters.user_id ||
                  filters.action ||
                  filters.target_type ||
                  filters.target_id ||
                  filters.start_date ||
                  filters.end_date) && (
                  <Button variant="outline" className="mt-4" onClick={clearFilters}>
                    Clear Filters
                  </Button>
                )}
              </div>
            ) : (
              <>
                <div className="rounded-md border overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Timestamp</TableHead>
                        <TableHead>User</TableHead>
                        <TableHead>Action</TableHead>
                        <TableHead>Resource Type</TableHead>
                        <TableHead>Resource ID</TableHead>
                        <TableHead>IP Address</TableHead>
                        <TableHead>Details</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {auditLogs.map((log) => {
                        const logUser = users.get(log.user_id);
                        return (
                          <TableRow key={log.id}>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                <Calendar className="h-4 w-4 text-muted-foreground" />
                                <span className="text-sm">{formatDate(log.timestamp)}</span>
                              </div>
                            </TableCell>
                            <TableCell>
                              <span className="text-sm">
                                {logUser?.email || log.user_id.slice(0, 8)}
                              </span>
                            </TableCell>
                            <TableCell>
                              <Badge variant="outline">{log.action}</Badge>
                            </TableCell>
                            <TableCell>
                              <Badge variant="secondary">{log.resource_type}</Badge>
                            </TableCell>
                            <TableCell>
                              <code className="text-xs">{log.resource_id.slice(0, 8)}...</code>
                            </TableCell>
                            <TableCell>
                              <span className="text-sm text-muted-foreground">
                                {log.ip_address || '-'}
                              </span>
                            </TableCell>
                            <TableCell>
                              {log.details ? (
                                <details className="text-xs">
                                  <summary className="cursor-pointer text-primary hover:underline">
                                    View Details
                                  </summary>
                                  <pre className="mt-2 p-2 bg-muted rounded-md overflow-x-auto max-w-md">
                                    {formatDetails(log.details)}
                                  </pre>
                                </details>
                              ) : (
                                <span className="text-sm text-muted-foreground">-</span>
                              )}
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>

                {/* Pagination */}
                {totalPages > 1 && (
                  <div className="flex items-center justify-between mt-4">
                    <div className="text-sm text-muted-foreground">
                      Page {currentPage} of {totalPages}
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
                        disabled={currentPage === 1 || isLoading}
                      >
                        Previous
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentPage((prev) => Math.min(totalPages, prev + 1))}
                        disabled={currentPage === totalPages || isLoading}
                      >
                        Next
                      </Button>
                    </div>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </DefaultLayout>
  );
};
