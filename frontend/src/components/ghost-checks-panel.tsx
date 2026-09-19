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

  // Parse the latest copy progress snapshot (persisted on the plan).
  // New snapshots are JSON ({copied,total,pct,elapsed}); older ones are
  // plain numbers. Completed ghost plans render a full bar even if the last
  // persisted snapshot was below 100 (cutover completes after the final
  // progress line).
  // Completed renders full; AwaitingCutover/Cleanup render near-full but
  // not 100 — gh-ost's copy is done but the migration isn't finished until
  // cutover is approved (and the backup dropped), so the bar shouldn't look
  // complete yet.
  const snapshot = parseGhostProgress(ghostProgress);
  const parsedProgress = snapshot?.pct ?? null;
  let progressValue = parsedProgress;
  if (planStatus === 'Completed') {
    progressValue = 100;
  } else if (planStatus === 'AwaitingCleanup') {
    progressValue = 99.99;
  } else if (planStatus === 'AwaitingCutover') {
    progressValue = 99.9;
  }
  const showProgress =
    progressValue != null &&
    (planStatus === 'InProgress' ||
      planStatus === 'AwaitingCutover' ||
      planStatus === 'AwaitingCleanup' ||
      (planStatus === 'Completed' && parsedProgress != null));

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
      {showProgress && progressValue != null && (
        <div className="px-6 pb-4">
          <div className="flex justify-between text-xs text-muted-foreground mb-1.5">
            <span>
              {planStatus === 'AwaitingCutover'
                ? 'Awaiting cutover approval'
                : planStatus === 'AwaitingCleanup'
                  ? 'Cutover complete — drop the backup to finish'
                  : planStatus === 'Completed'
                    ? 'Migration complete'
                    : 'Copying shadow table'}
              {snapshot != null &&
                snapshot.total > 0 &&
                ` — ${snapshot.copied.toLocaleString()}/${snapshot.total.toLocaleString()} rows`}
              {snapshot?.elapsed && ` · ${snapshot.elapsed} elapsed`}
            </span>
            <span>{progressValue.toFixed(1)}%</span>
          </div>
          <Progress value={progressValue} />
        </div>
      )}
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
