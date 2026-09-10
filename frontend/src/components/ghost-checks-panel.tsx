import React, { useState } from 'react';
import { CheckCircle2, XCircle, RefreshCw, Loader2 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import type { GhostReadinessReport } from '@/lib/api/types';
import { changePlanService } from '@/lib/api/services/changePlans';

interface GhostChecksPanelProps {
  changePlanId: string;
  report: GhostReadinessReport | null | undefined;
  ghostProgress?: string;
  planStatus?: string;
  onRefreshed?: (report: GhostReadinessReport) => void;
}

/**
 * PR-checks style panel showing ghost readiness checks with per-check
 * pass/fail, remediation detail, and re-run controls (all or individual).
 */
export const GhostChecksPanel: React.FC<GhostChecksPanelProps> = ({
  changePlanId,
  report,
  ghostProgress,
  planStatus,
  onRefreshed,
}) => {
  const [isRefreshingAll, setIsRefreshingAll] = useState(false);
  const [refreshingCheck, setRefreshingCheck] = useState<string | null>(null);

  // Parse the latest copy progress snapshot (".." persisted on the plan)
  const progressValue = ghostProgress ? parseFloat(ghostProgress) : null;
  const showProgress =
    progressValue != null &&
    (planStatus === 'InProgress' || planStatus === 'AwaitingCutover');

  if (!report && !showProgress) {
    return null;
  }

  const rerunAll = async () => {
    setIsRefreshingAll(true);
    try {
      const fresh = await changePlanService.rerunGhostChecks(changePlanId);
      onRefreshed?.(fresh);
    } finally {
      setIsRefreshingAll(false);
    }
  };

  const rerunOne = async (checkName: string) => {
    setRefreshingCheck(checkName);
    try {
      const fresh = await changePlanService.rerunGhostCheck(changePlanId, checkName);
      onRefreshed?.(fresh);
    } finally {
      setRefreshingCheck(null);
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-3">
        <CardTitle className="text-base font-medium">Ghost readiness checks</CardTitle>
        <Button
          variant="outline"
          size="sm"
          onClick={rerunAll}
          disabled={isRefreshingAll}
          className="gap-1.5"
        >
          {isRefreshingAll ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <RefreshCw className="h-3.5 w-3.5" />
          )}
          Re-run all checks
        </Button>
      </CardHeader>
      {showProgress && progressValue != null && (
        <div className="px-6 pb-4">
          <div className="flex justify-between text-xs text-muted-foreground mb-1.5">
            <span>{planStatus === 'AwaitingCutover' ? 'Awaiting cutover approval' : 'Copying shadow table'}</span>
            <span>{progressValue.toFixed(1)}%</span>
          </div>
          <Progress value={progressValue} />
        </div>
      )}
      <CardContent>
        <ul className="space-y-2.5">
          {report.checks.map((check) => (
            <li key={check.name} className="flex items-start gap-2.5">
              {check.passed ? (
                <CheckCircle2 className="h-4.5 w-4.5 mt-0.5 text-green-600 dark:text-green-400 shrink-0" />
              ) : (
                <XCircle className="h-4.5 w-4.5 mt-0.5 text-red-600 dark:text-red-400 shrink-0" />
              )}
              <div className="min-w-0 flex-1">
                <span className="text-sm font-medium">{check.name}</span>
                <p className="text-xs text-muted-foreground break-words">{check.detail}</p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => rerunOne(check.name)}
                disabled={refreshingCheck === check.name || isRefreshingAll}
                title={`Re-run ${check.name}`}
                className="shrink-0 h-7 px-2"
              >
                {refreshingCheck === check.name ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <RefreshCw className="h-3.5 w-3.5" />
                )}
              </Button>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
};
