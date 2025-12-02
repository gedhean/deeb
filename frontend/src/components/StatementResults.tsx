import React, { useState } from 'react';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { useStatementResultsSWR } from '@/hooks/api/useChangePlansSWR';
import type { StatementExecutionResult, StatementExecutionStatus } from '@/lib/api/types';
import { CheckCircle2, XCircle, SkipForward, Clock, ChevronDown, ChevronUp } from 'lucide-react';
import { cn } from '@/lib/utils';

interface StatementResultsProps {
  changePlanId: string;
  executionLogId: string;
}

const statusConfig: Record<StatementExecutionStatus, { icon: React.ReactNode; color: string }> = {
  Completed: {
    icon: <CheckCircle2 className="h-4 w-4" />,
    color: 'text-green-600 dark:text-green-400',
  },
  Failed: {
    icon: <XCircle className="h-4 w-4" />,
    color: 'text-red-600 dark:text-red-400',
  },
  Skipped: {
    icon: <SkipForward className="h-4 w-4" />,
    color: 'text-yellow-600 dark:text-yellow-400',
  },
};

function StatusBadge({ status }: { status: StatementExecutionStatus }) {
  const config = statusConfig[status];
  return (
    <Badge
      variant="outline"
      className={cn(
        'gap-1',
        status === 'Completed' &&
          'border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950/20 dark:text-green-400',
        status === 'Failed' &&
          'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950/20 dark:text-red-400',
        status === 'Skipped' &&
          'border-yellow-200 bg-yellow-50 text-yellow-700 dark:border-yellow-800 dark:bg-yellow-950/20 dark:text-yellow-400',
      )}
    >
      {config.icon}
      {status}
    </Badge>
  );
}

function formatDuration(ms: number): string {
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
}

function truncateSQL(sql: string, maxLength: number = 80): string {
  const singleLine = sql.replace(/\s+/g, ' ').trim();
  if (singleLine.length <= maxLength) {
    return singleLine;
  }
  return singleLine.substring(0, maxLength) + '...';
}

interface StatementResultItemProps {
  result: StatementExecutionResult;
}

function StatementResultItem({ result }: StatementResultItemProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const config = statusConfig[result.status];
  const hasPosition =
    result.start_line !== undefined &&
    result.start_column !== undefined &&
    result.end_line !== undefined;

  return (
    <div
      className={cn(
        'rounded-lg border p-3 space-y-2',
        result.status === 'Failed' && 'border-red-200 dark:border-red-800',
      )}
    >
      {/* Header row */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <span className={cn('shrink-0', config.color)}>{config.icon}</span>
          <span className="font-mono text-xs text-muted-foreground shrink-0">
            #{result.statement_index + 1}
          </span>
          <code className="text-sm truncate font-mono bg-muted px-1.5 py-0.5 rounded flex-1 min-w-0">
            {truncateSQL(result.statement_sql)}
          </code>
        </div>
        <StatusBadge status={result.status} />
      </div>

      {/* Metadata row */}
      <div className="flex items-center gap-4 text-xs text-muted-foreground">
        <div className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          <span>{formatDuration(result.execution_duration_ms)}</span>
        </div>
        {result.rows_affected !== undefined && (
          <span>
            {result.rows_affected.toLocaleString()} row{result.rows_affected !== 1 ? 's' : ''}{' '}
            affected
          </span>
        )}
        {hasPosition && (
          <span className="font-mono">
            L{result.start_line}:{result.start_column}
          </span>
        )}
      </div>

      {/* Error message */}
      {result.status === 'Failed' && result.error_message && (
        <div className="rounded-md border border-red-200 bg-red-50 p-2 dark:border-red-800 dark:bg-red-950/20">
          <p className="text-sm text-red-800 dark:text-red-200 whitespace-pre-wrap">
            {result.error_message}
          </p>
        </div>
      )}

      {/* Expandable SQL content */}
      {result.statement_sql.length > 80 && (
        <button
          type="button"
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-1 text-xs text-primary hover:underline"
        >
          {isExpanded ? (
            <>
              <ChevronUp className="h-3 w-3" />
              Hide full SQL
            </>
          ) : (
            <>
              <ChevronDown className="h-3 w-3" />
              Show full SQL
            </>
          )}
        </button>
      )}
      {isExpanded && (
        <pre className="mt-2 p-3 bg-muted rounded-md overflow-x-auto text-xs">
          <code>{result.statement_sql}</code>
        </pre>
      )}
    </div>
  );
}

export const StatementResults: React.FC<StatementResultsProps> = ({
  changePlanId,
  executionLogId,
}) => {
  const { statementResults, isLoading, error } = useStatementResultsSWR(
    changePlanId,
    executionLogId,
  );

  if (isLoading) {
    return (
      <div className="space-y-2 mt-3">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20 mt-3">
        <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
          <XCircle className="h-4 w-4" />
          <span className="text-sm">{error}</span>
        </div>
      </div>
    );
  }

  const results = statementResults?.statement_results || [];

  if (results.length === 0) {
    return (
      <p className="text-sm text-muted-foreground mt-3">
        No statement execution details available.
      </p>
    );
  }

  // Count statuses
  const completedCount = results.filter((r) => r.status === 'Completed').length;
  const failedCount = results.filter((r) => r.status === 'Failed').length;
  const skippedCount = results.filter((r) => r.status === 'Skipped').length;

  return (
    <Accordion type="single" collapsible className="w-full mt-3">
      <AccordionItem value="statement-results" className="border rounded-lg">
        <AccordionTrigger className="px-4 hover:no-underline">
          <div className="flex items-center gap-3">
            <span className="text-sm font-medium">Statement Execution Details</span>
            <div className="flex items-center gap-2">
              {completedCount > 0 && (
                <Badge variant="outline" className="bg-green-50 text-green-700 text-xs">
                  {completedCount} completed
                </Badge>
              )}
              {failedCount > 0 && (
                <Badge variant="outline" className="bg-red-50 text-red-700 text-xs">
                  {failedCount} failed
                </Badge>
              )}
              {skippedCount > 0 && (
                <Badge variant="outline" className="bg-yellow-50 text-yellow-700 text-xs">
                  {skippedCount} skipped
                </Badge>
              )}
            </div>
          </div>
        </AccordionTrigger>
        <AccordionContent className="px-4 pb-4">
          <div className="space-y-2">
            {results.map((result) => (
              <StatementResultItem key={result.id} result={result} />
            ))}
          </div>
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  );
};

export default StatementResults;
