import React, { useState } from 'react';
import { CheckCircle2, XCircle, RefreshCw, Loader2 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import type { GhostReadinessReport } from '@/lib/api/types';
import { changePlanService } from '@/lib/api/services/changePlans';

interface GhostProgressSnapshot {
  copied: number;
  total: number;
  pct: number;
  elapsed?: string;
}

// parseGhostProgress accepts both JSON snapshots ({copied,total,pct,elapsed})
// and legacy plain-number values persisted by earlier backend versions.
function parseGhostProgress(raw: string | null | undefined): GhostProgressSnapshot | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    if (typeof parsed === 'object' && parsed !== null && 'pct' in parsed) {
      return parsed as GhostProgressSnapshot;
    }
  } catch {
    // not JSON — fall through to plain number
  }
  const num = parseFloat(raw);
  return Number.isNaN(num) ? null : ({ copied: 0, total: 0, pct: num } as GhostProgressSnapshot);
}

interface GhostProgressBarProps {
  ghostProgress?: string;
  planStatus?: string;
}

// useGhostProgress parses the persisted snapshot and computes the display
// percentage for the plan's current status. Shared by the standalone
// GhostProgressBar (rendered above Basic Information) and the checks panel.
function useGhostProgress(ghostProgress?: string, planStatus?: string) {
  const snapshot = parseGhostProgress(ghostProgress);
  const parsedProgress = snapshot?.pct ?? null;
  let progressValue = parsedProgress;
  if (planStatus === 'Completed') {
    progressValue = 100;
  } else if (planStatus === 'AwaitingCleanup') {
    progressValue = 99.99;
  } else if (planStatus === 'AwaitingCutover') {
    progressValue = 99.9;
  } else if (planStatus === 'InProgress' && progressValue == null) {
    // Copy hasn't emitted its first status line yet (binlog connect, table
    // setup) — render an empty bar instead of hiding it entirely.
    progressValue = 0;
  }
  const showProgress =
    progressValue != null &&
    (planStatus === 'InProgress' ||
      planStatus === 'AwaitingCutover' ||
      planStatus === 'AwaitingCleanup' ||
      (planStatus === 'Completed' && parsedProgress != null));
  return { snapshot, progressValue, showProgress };
}

function progressLabel(planStatus?: string): string {
  switch (planStatus) {
    case 'AwaitingCutover':
      return 'Awaiting cutover approval';
    case 'AwaitingCleanup':
      return 'Cutover complete — drop the backup to finish';
    case 'Completed':
      return 'Migration complete';
    default:
      return 'Copying shadow table';
  }
}

// GhostProgressBar renders just the live copy progress — placed near the top
// of the change plan detail page, separate from the checks panel.
export const GhostProgressBar: React.FC<GhostProgressBarProps> = ({
  ghostProgress,
  planStatus,
}) => {
  const { snapshot, progressValue, showProgress } = useGhostProgress(ghostProgress, planStatus);
  if (!showProgress || progressValue == null) {
    return null;
  }
  return (
    <Card className="w-full">
      <CardContent className="pt-5 pb-4">
        <div className="flex justify-between text-xs text-muted-foreground mb-1.5">
          <span>
            {progressLabel(planStatus)}
            {snapshot != null &&
              snapshot.total > 0 &&
              ` — ${snapshot.copied.toLocaleString()}/${snapshot.total.toLocaleString()} rows`}
            {snapshot?.elapsed && ` · ${snapshot.elapsed} elapsed`}
          </span>
          <span>
            {planStatus === 'AwaitingCleanup'
              ? '99.99'
              : planStatus === 'AwaitingCutover'
                ? '99.9'
                : progressValue.toFixed(1)}
            %
          </span>
        </div>
        <Progress
          value={progressValue}
          // Blue while copying/gated (InProgress, AwaitingCutover,
          // AwaitingCleanup); green only when fully complete.
          indicatorClassName={planStatus === 'Completed' ? 'bg-emerald-500' : 'bg-blue-500'}
        />
      </CardContent>
    </Card>
  );
};

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

  const { showProgress } = useGhostProgress(ghostProgress, planStatus);

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
        <CardTitle className="text-base font-medium">Online Migrator readiness checks</CardTitle>
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
      <CardContent>
        <ul className="space-y-2.5">
          {report?.checks.map((check) => (
            <li key={check.name} className="flex items-start gap-2.5">
              {check.passed ? (
                <CheckCircle2 className="h-4.5 w-4.5 mt-0.5 text-green-600 dark:text-green-400 shrink-0" />
              ) : (
                <XCircle className="h-4.5 w-4.5 mt-0.5 text-red-600 dark:text-red-400 shrink-0" />
              )}
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5">
                  <span className="text-sm font-medium">{check.name}</span>
                </div>
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
