import React, { useState, useMemo } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { PolicyFindingsSummary, PolicyFindingItem, ChangePlanStatus } from '@/lib/api/types';
import { useTriggerSQLReview } from '@/hooks/api/useChangePlansSWR';
import { AlertTriangle, CheckCircle2, Info, Loader2, RefreshCw, Filter } from 'lucide-react';

interface SQLReviewResultsProps {
  changePlanId: string;
  findings?: PolicyFindingsSummary;
  changePlanStatus?: ChangePlanStatus;
  onRefresh?: () => void;
}

export const SQLReviewResults: React.FC<SQLReviewResultsProps> = ({
  changePlanId,
  findings,
  changePlanStatus,
  onRefresh,
}) => {
  // Use SWR mutation hook for consistent cache management
  const { triggerSQLReview, isLoading: isTriggeringReview } = useTriggerSQLReview();

  const [error, setError] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [severityFilter, setSeverityFilter] = useState<string>('all');
  const [ruleNameFilter, setRuleNameFilter] = useState<string>('');
  const [validatorTypeFilter, setValidatorTypeFilter] = useState<string>('');

  const handleTriggerReview = async () => {
    setError('');

    // Trigger SQL Review
    const success = await triggerSQLReview(changePlanId);

    if (success) {
      // Refresh the change plan data if callback provided
      if (onRefresh) {
        onRefresh();
      }
    } else {
      setError('Failed to trigger SQL Review. Please try again.');
    }
  };

  // Apply filters to findings (must be before early returns to satisfy React Hooks rules)
  const filteredFindings = useMemo(() => {
    if (!findings) {
      return {
        errors: [],
        warnings: [],
        infos: [],
        counts: { errors: 0, warnings: 0, infos: 0 },
      };
    }

    const filterItems = (items: PolicyFindingItem[]) => {
      return items.filter((item) => {
        // Filter by rule name
        if (
          ruleNameFilter &&
          !item.rule_name.toLowerCase().includes(ruleNameFilter.toLowerCase())
        ) {
          return false;
        }
        // Filter by validator type
        if (
          validatorTypeFilter &&
          item.validator_type &&
          !item.validator_type.toLowerCase().includes(validatorTypeFilter.toLowerCase())
        ) {
          return false;
        }
        return true;
      });
    };

    const filteredErrors = filterItems(findings.errors);
    const filteredWarnings = filterItems(findings.warnings);
    const filteredInfos = filterItems(findings.infos);

    // Apply severity filter
    let displayedErrors = filteredErrors;
    let displayedWarnings = filteredWarnings;
    let displayedInfos = filteredInfos;

    if (severityFilter === 'errors') {
      displayedWarnings = [];
      displayedInfos = [];
    } else if (severityFilter === 'warnings') {
      displayedErrors = [];
      displayedInfos = [];
    } else if (severityFilter === 'infos') {
      displayedErrors = [];
      displayedWarnings = [];
    }

    return {
      errors: displayedErrors,
      warnings: displayedWarnings,
      infos: displayedInfos,
      counts: {
        errors: displayedErrors.length,
        warnings: displayedWarnings.length,
        infos: displayedInfos.length,
      },
    };
  }, [findings, severityFilter, ruleNameFilter, validatorTypeFilter]);

  // Check if any filters are active
  const hasActiveFilters =
    severityFilter !== 'all' || ruleNameFilter !== '' || validatorTypeFilter !== '';

  // If no findings provided, show appropriate message or loading state
  if (!findings) {
    // Show skeleton only when user triggers the review
    if (isTriggeringReview) {
      return (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>SQL Review Results</CardTitle>
                <CardDescription>SQL Review results will appear here</CardDescription>
              </div>
              <Button variant="outline" size="sm" disabled={true}>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Running Review...
              </Button>
            </div>
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

    // Show message when change plan is Draft (not submitted for review yet)
    if (changePlanStatus === 'Draft') {
      return (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>SQL Review Results</CardTitle>
                <CardDescription>SQL Review results will appear here</CardDescription>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleTriggerReview}
                disabled={isTriggeringReview}
              >
                <RefreshCw className="mr-2 h-4 w-4" />
                Run Review
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-left text-muted-foreground">
              <p className="text-sm font-medium">No review results yet</p>
              <p className="text-xs mt-1">
                SQL Review results will appear here when you submit this change plan for review.
              </p>
            </div>
          </CardContent>
        </Card>
      );
    }

    // Show message when change plan is submitted but no review results yet
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>SQL Review Results</CardTitle>
              <CardDescription>SQL Review results will appear here</CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={handleTriggerReview}
              disabled={isTriggeringReview}
            >
              <RefreshCw className="mr-2 h-4 w-4" />
              Run Review
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            <p className="text-xs mt-1">
              SQL Review results will appear here when the change plan is submitted for review.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  const { errors, warnings, infos, counts } = filteredFindings;
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
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowFilters(!showFilters)}
              className={hasActiveFilters ? 'border-blue-500 text-blue-600' : ''}
            >
              <Filter className="mr-2 h-4 w-4" />
              {hasActiveFilters ? 'Filters Active' : 'Filter'}
            </Button>
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
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Filter controls */}
        {showFilters && (
          <div className="p-4 rounded-md border bg-muted/50">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label htmlFor="severity-filter" className="text-xs font-medium">
                  Severity
                </Label>
                <Select value={severityFilter} onValueChange={setSeverityFilter}>
                  <SelectTrigger id="severity-filter" className="h-9">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Severities</SelectItem>
                    <SelectItem value="errors">Errors Only</SelectItem>
                    <SelectItem value="warnings">Warnings Only</SelectItem>
                    <SelectItem value="infos">Info Only</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="rule-name-filter" className="text-xs font-medium">
                  Rule Name
                </Label>
                <Input
                  id="rule-name-filter"
                  placeholder="Filter by rule name..."
                  value={ruleNameFilter}
                  onChange={(e) => setRuleNameFilter(e.target.value)}
                  className="h-9 text-sm"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="validator-type-filter" className="text-xs font-medium">
                  Validator Type
                </Label>
                <Input
                  id="validator-type-filter"
                  placeholder="Filter by validator..."
                  value={validatorTypeFilter}
                  onChange={(e) => setValidatorTypeFilter(e.target.value)}
                  className="h-9 text-sm"
                />
              </div>
            </div>

            {hasActiveFilters && (
              <div className="mt-3 flex items-center justify-between">
                <span className="text-xs text-muted-foreground">
                  Showing {counts.errors + counts.warnings + counts.infos} of{' '}
                  {findings.counts.errors + findings.counts.warnings + findings.counts.infos}{' '}
                  findings
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setSeverityFilter('all');
                    setRuleNameFilter('');
                    setValidatorTypeFilter('');
                  }}
                  className="h-7 text-xs"
                >
                  Clear Filters
                </Button>
              </div>
            )}
          </div>
        )}
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
