import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import type { PolicyFindingsSummary, PolicyFindingItem } from '@/lib/api/types';
import { changePlanService } from '@/lib/api/services/changePlans';
import { APIError } from '@/lib/api/types';
import { AlertTriangle, CheckCircle2, Info, Loader2, RefreshCw } from 'lucide-react';

interface SQLReviewResultsProps {
  changePlanId: string;
  findings?: PolicyFindingsSummary;
  onRefresh?: () => void;
}

export const SQLReviewResults: React.FC<SQLReviewResultsProps> = ({
  changePlanId,
  findings,
  onRefresh,
}) => {
  const [isTriggeringReview, setIsTriggeringReview] = useState(false);
  const [error, setError] = useState('');

  const handleTriggerReview = async () => {
    try {
      setIsTriggeringReview(true);
      setError('');

      // Trigger SQL Review
      await changePlanService.triggerSQLReview(changePlanId);

      // Refresh the change plan data if callback provided
      if (onRefresh) {
        onRefresh();
      }
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message);
      } else {
        setError('Failed to trigger SQL Review. Please try again.');
      }
      console.error('Failed to trigger SQL Review:', err);
    } finally {
      setIsTriggeringReview(false);
    }
  };

  // If no findings provided, show loading or empty state
  if (!findings) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>SQL Review Results</CardTitle>
          <CardDescription>SQL Review results will appear here</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        </CardContent>
      </Card>
    );
  }

  const { errors, warnings, infos, counts } = findings;
  const hasErrors = errors.length > 0;
  const hasWarnings = warnings.length > 0;
  const hasInfos = infos.length > 0;
  const hasAnyFindings = hasErrors || hasWarnings || hasInfos;

  // Render a single finding item
  const renderFinding = (finding: PolicyFindingItem, severity: 'error' | 'warning' | 'info') => {
    const iconClass =
      severity === 'error'
        ? 'text-red-600 dark:text-red-400'
        : severity === 'warning'
          ? 'text-yellow-600 dark:text-yellow-400'
          : 'text-blue-600 dark:text-blue-400';

    const bgClass =
      severity === 'error'
        ? 'bg-red-50 dark:bg-red-950/20 border-red-200 dark:border-red-800'
        : severity === 'warning'
          ? 'bg-yellow-50 dark:bg-yellow-950/20 border-yellow-200 dark:border-yellow-800'
          : 'bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-800';

    return (
      <div
        key={`${finding.rule_name}-${finding.line_number || 0}`}
        className={`p-3 rounded-md border ${bgClass}`}
      >
        <div className="flex items-start gap-3">
          <div className={`mt-0.5 ${iconClass}`}>
            {severity === 'error' ? (
              <AlertTriangle className="h-4 w-4" />
            ) : severity === 'warning' ? (
              <AlertTriangle className="h-4 w-4" />
            ) : (
              <Info className="h-4 w-4" />
            )}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <span className="text-sm font-medium">{finding.rule_name}</span>
              {finding.line_number && (
                <Badge variant="outline" className="text-xs">
                  Line {finding.line_number}
                  {finding.column_number && `, Col ${finding.column_number}`}
                </Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground">{finding.message}</p>
            {finding.validator_type && (
              <p className="text-xs text-muted-foreground mt-1">
                Validator: {finding.validator_type}
              </p>
            )}
          </div>
        </div>
      </div>
    );
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>SQL Review Results</CardTitle>
            <CardDescription>Automated SQL review findings grouped by severity</CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={handleTriggerReview}
            disabled={isTriggeringReview}
          >
            {isTriggeringReview ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Running Review...
              </>
            ) : (
              <>
                <RefreshCw className="mr-2 h-4 w-4" />
                Run Review
              </>
            )}
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Error message */}
        {error && (
          <div className="p-3 rounded-md border border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/20">
            <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
              <AlertTriangle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          </div>
        )}

        {/* Summary badges */}
        <div className="flex items-center gap-3">
          {counts.errors > 0 && (
            <Badge
              variant="destructive"
              className="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
            >
              {counts.errors} Error{counts.errors !== 1 ? 's' : ''}
            </Badge>
          )}
          {counts.warnings > 0 && (
            <Badge className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
              {counts.warnings} Warning{counts.warnings !== 1 ? 's' : ''}
            </Badge>
          )}
          {counts.infos > 0 && (
            <Badge
              variant="outline"
              className="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
            >
              {counts.infos} Info{counts.infos !== 1 ? 's' : ''}
            </Badge>
          )}
          {!hasAnyFindings && (
            <div className="flex items-center gap-2 text-green-600 dark:text-green-400">
              <CheckCircle2 className="h-4 w-4" />
              <span className="text-sm font-medium">All checks passed</span>
            </div>
          )}
        </div>

        {/* Error findings */}
        {hasErrors && (
          <div className="space-y-2">
            <h3 className="text-sm font-semibold text-red-600 dark:text-red-400 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              Errors ({errors.length})
            </h3>
            <div className="space-y-2">
              {errors.map((finding) => renderFinding(finding, 'error'))}
            </div>
          </div>
        )}

        {/* Warning findings */}
        {hasWarnings && (
          <div className="space-y-2">
            <h3 className="text-sm font-semibold text-yellow-600 dark:text-yellow-400 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              Warnings ({warnings.length})
            </h3>
            <div className="space-y-2">
              {warnings.map((finding) => renderFinding(finding, 'warning'))}
            </div>
          </div>
        )}

        {/* Info findings */}
        {hasInfos && (
          <div className="space-y-2">
            <h3 className="text-sm font-semibold text-blue-600 dark:text-blue-400 flex items-center gap-2">
              <Info className="h-4 w-4" />
              Informational ({infos.length})
            </h3>
            <div className="space-y-2">
              {infos.map((finding) => renderFinding(finding, 'info'))}
            </div>
          </div>
        )}

        {/* No findings message */}
        {!hasAnyFindings && (
          <div className="text-center py-8 text-muted-foreground">
            <CheckCircle2 className="h-12 w-12 mx-auto mb-2 text-green-600 dark:text-green-400" />
            <p className="text-sm font-medium">No issues found</p>
            <p className="text-xs mt-1">All SQL Review checks passed successfully</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
