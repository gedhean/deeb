import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { changePlanService } from '@/lib/api/services/changePlans';
import { usersService } from '@/lib/api/services/users';
import { APIError } from '@/lib/api/types';
import type { ChangePlan, ExecutionLog, User, ChangePlanStatus } from '@/lib/api/types';
import { ChangePlanStatusBadge } from '@/components/ChangePlanStatusBadge';
import {
  CheckCircle2,
  XCircle,
  Clock,
  Database as DatabaseIcon,
  Calendar,
  User as UserIcon,
} from 'lucide-react';

interface ExecutionResultsProps {
  changePlan: ChangePlan;
}

export const ExecutionResults: React.FC<ExecutionResultsProps> = ({ changePlan }) => {
  const [executionLogs, setExecutionLogs] = useState<ExecutionLog[]>([]);
  const [executors, setExecutors] = useState<Map<string, User>>(new Map());
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    // Only load execution logs if change plan has been executed
    if (
      changePlan.status === 'Completed' ||
      changePlan.status === 'Failed' ||
      changePlan.executed_at
    ) {
      loadExecutionLogs();
    } else {
      setIsLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [changePlan.id, changePlan.status, changePlan.executed_at]);

  const loadExecutionLogs = async () => {
    try {
      setIsLoading(true);
      setError('');

      // Load execution log history
      const history = await changePlanService.getExecutionLogHistory(changePlan.id);
      setExecutionLogs(history.logs || []);

      // Load executor information
      if (history.logs && history.logs.length > 0) {
        const executorIds = [...new Set(history.logs.map((log) => log.executor_id))];
        const usersRes = await usersService.listUsers({ limit: 100 });

        const executorsMap = new Map<string, User>();
        executorIds.forEach((executorId) => {
          const executor = usersRes.items.find((u) => u.id === executorId);
          if (executor) {
            executorsMap.set(executorId, executor);
          }
        });

        setExecutors(executorsMap);
      }
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message || 'Failed to load execution logs.');
      } else {
        setError('Failed to load execution logs. Please try again.');
      }
      console.error('Failed to load execution logs:', err);
    } finally {
      setIsLoading(false);
    }
  };

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

  const formatDuration = (ms: number): string => {
    if (ms < 1000) {
      return `${ms}ms`;
    }
    const seconds = Math.floor(ms / 1000);
    const milliseconds = ms % 1000;
    if (seconds < 60) {
      return milliseconds > 0 ? `${seconds}s ${milliseconds}ms` : `${seconds}s`;
    }
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}m ${remainingSeconds}s`;
  };

  // Determine if there's execution data to show
  const hasExecutionData =
    changePlan.status === 'Completed' ||
    changePlan.status === 'Failed' ||
    changePlan.executed_at ||
    executionLogs.length > 0;

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Execution Results</CardTitle>
          <CardDescription>Execution status and history for this change plan</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-20 w-full" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!hasExecutionData) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Execution Results</CardTitle>
          <CardDescription>Execution status and history for this change plan</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            This change plan has not been executed yet.
          </p>
        </CardContent>
      </Card>
    );
  }

  // Get latest execution status from change plan or latest log
  const latestLog = executionLogs.length > 0 ? executionLogs[0] : null;
  const executionStatus =
    changePlan.status === 'Completed' || changePlan.status === 'Failed'
      ? changePlan.status
      : latestLog?.status || null;

  const isCompleted = executionStatus === 'Completed';
  const isFailed = executionStatus === 'Failed';

  return (
    <Card>
      <CardHeader>
        <CardTitle>Execution Results</CardTitle>
        <CardDescription>Execution status and history for this change plan</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Current Execution Status */}
        {executionStatus && (
          <div className="rounded-lg border p-4 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div
                  className={
                    isCompleted
                      ? 'text-green-600 dark:text-green-400'
                      : isFailed
                        ? 'text-red-600 dark:text-red-400'
                        : 'text-muted-foreground'
                  }
                >
                  {isCompleted ? (
                    <CheckCircle2 className="h-5 w-5" />
                  ) : isFailed ? (
                    <XCircle className="h-5 w-5" />
                  ) : null}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">Status</span>
                    {executionStatus && (
                      <ChangePlanStatusBadge status={executionStatus as ChangePlanStatus} />
                    )}
                  </div>
                </div>
              </div>
            </div>

            {/* Execution Details */}
            <div className="grid gap-3 md:grid-cols-2">
              {changePlan.executed_at && (
                <div className="flex items-center gap-2 text-sm">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Executed:</span>
                  <span className="font-medium">{formatDate(changePlan.executed_at)}</span>
                </div>
              )}

              {changePlan.execution_duration_ms !== undefined && (
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Duration:</span>
                  <span className="font-medium">
                    {formatDuration(changePlan.execution_duration_ms)}
                  </span>
                </div>
              )}

              {changePlan.affected_rows !== undefined && (
                <div className="flex items-center gap-2 text-sm">
                  <DatabaseIcon className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Rows Affected:</span>
                  <span className="font-medium">{changePlan.affected_rows.toLocaleString()}</span>
                </div>
              )}
            </div>

            {/* Error Message */}
            {isFailed && changePlan.error_message && (
              <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
                <div className="flex items-start gap-2">
                  <XCircle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5" />
                  <div className="flex-1">
                    <p className="text-sm font-medium text-red-800 dark:text-red-200">
                      Execution Error
                    </p>
                    <p className="text-sm text-red-700 dark:text-red-300 mt-1 whitespace-pre-wrap">
                      {changePlan.error_message}
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Execution Log History */}
        {executionLogs.length > 0 && (
          <Accordion type="single" collapsible className="w-full">
            <AccordionItem value="execution-history" className="border-none">
              <AccordionTrigger className="px-0 hover:no-underline">
                <h3 className="text-sm font-semibold">Execution History</h3>
              </AccordionTrigger>
              <AccordionContent>
                <div className="space-y-3 mt-4">
                  {executionLogs.map((log) => {
                    const executor = executors.get(log.executor_id);
                    const logIsCompleted = log.status === 'Completed';
                    const logIsFailed = log.status === 'Failed';

                    return (
                      <div key={log.id} className="rounded-lg border p-4 space-y-2">
                        <div className="flex items-start justify-between">
                          <div className="flex items-center gap-2">
                            {logIsCompleted ? (
                              <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                            ) : logIsFailed ? (
                              <XCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
                            ) : null}
                            {log.status && (
                              <ChangePlanStatusBadge status={log.status as ChangePlanStatus} />
                            )}
                          </div>
                          <span className="text-xs text-muted-foreground">
                            {formatDate(log.completed_at || log.started_at)}
                          </span>
                        </div>

                        <div className="grid gap-2 text-sm md:grid-cols-2">
                          <div className="flex items-center gap-2">
                            <Clock className="h-4 w-4 text-muted-foreground" />
                            <span className="text-muted-foreground">Duration:</span>
                            <span className="font-medium">
                              {formatDuration(log.execution_duration_ms)}
                            </span>
                          </div>

                          {log.rows_affected !== undefined && (
                            <div className="flex items-center gap-2">
                              <DatabaseIcon className="h-4 w-4 text-muted-foreground" />
                              <span className="text-muted-foreground">Rows Affected:</span>
                              <span className="font-medium">
                                {log.rows_affected.toLocaleString()}
                              </span>
                            </div>
                          )}

                          {executor && (
                            <div className="flex items-center gap-2">
                              <UserIcon className="h-4 w-4 text-muted-foreground" />
                              <span className="text-muted-foreground">Executor:</span>
                              <span className="font-medium">{executor.email}</span>
                            </div>
                          )}

                          {log.database_name && (
                            <div className="flex items-center gap-2">
                              <DatabaseIcon className="h-4 w-4 text-muted-foreground" />
                              <span className="text-muted-foreground">Database:</span>
                              <span className="font-medium">{log.database_name}</span>
                            </div>
                          )}
                        </div>

                        {logIsFailed && log.error_message && (
                          <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20 mt-2">
                            <p className="text-sm text-red-800 dark:text-red-200 whitespace-pre-wrap">
                              {log.error_message}
                            </p>
                          </div>
                        )}

                        {log.sql_content && (
                          <details className="mt-2">
                            <summary className="text-sm font-medium cursor-pointer hover:text-primary">
                              View Executed SQL
                            </summary>
                            <pre className="mt-2 p-3 bg-muted rounded-md overflow-x-auto text-xs">
                              <code>{log.sql_content}</code>
                            </pre>
                          </details>
                        )}
                      </div>
                    );
                  })}
                </div>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        )}

        {error && (
          <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
            <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
              <XCircle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
