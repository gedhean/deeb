import React from 'react';
import { CheckCircle2, XCircle, RefreshCw } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import type { GhostReadinessReport } from '@/lib/api/types';

interface GhostChecksPanelProps {
  report: GhostReadinessReport | null | undefined;
  onRefresh?: () => void;
  isRefreshing?: boolean;
}

/**
 * PR-checks style panel showing ghost readiness checks with per-check
 * pass/fail and remediation detail (mirrors GitHub PR check runs).
 */
export const GhostChecksPanel: React.FC<GhostChecksPanelProps> = ({
  report,
  onRefresh,
  isRefreshing,
}) => {
  if (!report) {
    return null;
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
        <CardTitle className="text-base font-medium">Ghost readiness checks</CardTitle>
        {onRefresh && (
          <Button
            variant="outline"
            size="sm"
            onClick={onRefresh}
            disabled={isRefreshing}
            className="gap-1.5"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isRefreshing ? 'animate-spin' : ''}`} />
            Re-run checks
          </Button>
        )}
      </CardHeader>
      <CardContent>
        <ul className="space-y-2.5">
          {report.checks.map((check) => (
            <li key={check.name} className="flex items-start gap-2.5">
              {check.passed ? (
                <CheckCircle2 className="h-4.5 w-4.5 mt-0.5 text-green-600 dark:text-green-400 shrink-0" />
              ) : (
                <XCircle className="h-4.5 w-4.5 mt-0.5 text-red-600 dark:text-red-400 shrink-0" />
              )}
              <div className="min-w-0">
                <span className="text-sm font-medium">{check.name}</span>
                <p className="text-xs text-muted-foreground break-words">{check.detail}</p>
              </div>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
};
