import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import {
  useUpdateChangePlanSWR,
  useApproveChangePlan,
  useCloseChangePlan,
  useExecuteChangePlan,
} from '@/hooks/api/useChangePlansSWR';
import { changePlanService } from '@/lib/api/services/changePlans';
import type { ChangePlan, GhostReadinessReport } from '@/lib/api/types';
import {
  Send,
  CheckCircle2,
  XCircle,
  Play,
  Loader2,
  AlertTriangle,
  MoreVertical,
  Trash2,
} from 'lucide-react';
import { showSuccessToast } from '@/lib/errorHandling';

interface ChangePlanActionsProps {
  changePlan: ChangePlan;
  onSuccess?: () => void;
  canApprove: boolean;
  canReject: boolean;
  canExecute: boolean;
}

type ActionType =
  | 'submit'
  | 'approve'
  | 'close'
  | 'execute'
  | 'retry'
  | 'approve-cutover'
  | 'abort'
  | 'ghost-rollback'
  | 'ghost-drop-backup';

export const ChangePlanActions: React.FC<ChangePlanActionsProps> = ({
  changePlan,
  onSuccess,
  canApprove,
  canReject,
  canExecute,
}) => {
  // Use SWR mutation hooks for consistent cache management
  const { isLoading: isUpdating } = useUpdateChangePlanSWR();
  const { approveChangePlan, isLoading: isApproving } = useApproveChangePlan();
  const { closeChangePlan, isLoading: isClosing } = useCloseChangePlan();
  const { executeChangePlan, isLoading: isExecuting } = useExecuteChangePlan();

  const [activeAction, setActiveAction] = useState<ActionType | null>(null);
  const [comments, setComments] = useState('');
  const [error, setError] = useState('');
  const [ghostChecks, setGhostChecks] = useState<GhostReadinessReport | null>(null);
  const [executionStatus, setExecutionStatus] = useState<'idle' | 'executing' | 'success'>('idle');

  // Computed combined loading state
  const isSubmitting = isUpdating || isApproving || isClosing || isExecuting;

  // Reset state when dialog closes
  const handleDialogClose = () => {
    setComments('');
    setError('');
    setExecutionStatus('idle');
    setActiveAction(null);
  };

  // Ghost executor actions
  const handleApproveCutover = async () => {
    setError('');
    try {
      await changePlanService.approveGhostCutover(changePlan.id);
      showSuccessToast('Cutover approved');
      handleDialogClose();
      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve cutover');
    }
  };

  const handleAbort = async () => {
    setError('');
    try {
      await changePlanService.abortGhostExecution(changePlan.id);
      showSuccessToast('Abort requested');
      handleDialogClose();
      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to abort migration');
    }
  };

  const handleGhostRollback = async () => {
    setError('');
    try {
      await changePlanService.rollbackGhostPlan(changePlan.id);
      showSuccessToast('Plan rolled back');
      handleDialogClose();
      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to roll back plan');
    }
  };

  const handleGhostDropBackup = async () => {
    setError('');
    try {
      await changePlanService.dropGhostBackupTable(changePlan.id);
      showSuccessToast('Backup table dropped');
      handleDialogClose();
      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to drop backup table');
    }
  };

  const isGhostPlan = changePlan.execution_backend === 'ghost';

  // ghost_checks may arrive as a JSON string (TEXT column) — parse once
  const planGhostChecks: GhostReadinessReport | null =
    typeof changePlan.ghost_checks === 'string'
      ? (() => {
          try {
            return JSON.parse(changePlan.ghost_checks) as GhostReadinessReport;
          } catch {
            return null;
          }
        })()
      : (changePlan.ghost_checks ?? null);

  // Determine primary action based on status
  const getPrimaryAction = (): ActionType | null => {
    switch (changePlan.status) {
      case 'Draft':
        return 'submit';
      case 'Open':
        return canApprove ? 'approve' : null;
      case 'Approved':
        return canExecute ? 'execute' : null;
      case 'AwaitingCutover':
        return isGhostPlan && canExecute ? 'approve-cutover' : null;
      case 'Failed':
        return canApprove ? 'retry' : null;
      default:
        return null;
    }
  };

  // Get available secondary actions
  const getSecondaryActions = (): ActionType[] => {
    const actions: ActionType[] = [];

    switch (changePlan.status) {
      case 'Open':
        if (canReject) {
          actions.push('close');
        }
        break;
      case 'Approved':
        if (canReject) {
          actions.push('close');
        }
        break;
      case 'InProgress':
      case 'AwaitingCutover':
        if (isGhostPlan && canExecute) {
          actions.push('abort');
        }
        break;
      case 'AwaitingCleanup':
        // Cutover done, live on new schema; dropping the backup completes
        // the plan. Rollback stays available (swap back while the backup
        // exists) — but post-cutover writes are NOT in the backup.
        if (isGhostPlan && canExecute) {
          actions.push('ghost-drop-backup', 'ghost-rollback');
        }
        break;
      case 'Completed':
        // Legacy completed plans may still hold a backup to clean up.
        if (isGhostPlan && canExecute) {
          actions.push('ghost-rollback', 'ghost-drop-backup');
        }
        break;
      case 'Failed':
        if (canReject) {
          actions.push('close');
        }
        // Ghost plans that failed may still have a backup table to clean up
        // (e.g. failed after cutover or after a rollback swap).
        if (isGhostPlan && canExecute) {
          actions.push('ghost-drop-backup');
        }
        break;
    }

    return actions;
  };

  // Handle submit for review
  const handleSubmit = async () => {
    setError('');
    setGhostChecks(null);

    try {
      await changePlanService.updateChangePlan(changePlan.id, { status: 'Open' });
      handleDialogClose();
      showSuccessToast('Change plan submitted for review. SQL Review will run automatically.');
      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      // Ghost readiness failures return a structured per-check report
      const response = (
        err as { response?: { ghost_checks?: GhostReadinessReport; error?: string } }
      )?.response;
      if (response?.ghost_checks) {
        setGhostChecks(response.ghost_checks);
        setError('Online Migrator readiness checks failed — see details below.');
      } else {
        setError(
          `Failed to submit change plan for review: ${response?.error || (err instanceof Error ? err.message : 'Unknown error')}`,
        );
      }
    }
  };

  // Handle approve
  const handleApprove = async () => {
    if (!canApprove) {
      setError('You do not have permission to approve change plans.');
      return;
    }

    setError('');

    const success = await approveChangePlan(changePlan.id, {
      comments: comments.trim() || undefined,
    });

    if (success) {
      handleDialogClose();
      showSuccessToast(
        changePlan.status === 'Failed'
          ? 'Change plan approved for retry. You can now execute it again.'
          : 'Change plan approved successfully',
      );

      if (onSuccess) {
        onSuccess();
      }
    } else {
      setError('Failed to approve change plan');
    }
  };

  // Handle close
  const handleClose = async () => {
    if (!canReject) {
      setError('You do not have permission to close change plans.');
      return;
    }

    setError('');

    const success = await closeChangePlan(changePlan.id, {
      comments: comments.trim() || undefined,
    });

    if (success) {
      handleDialogClose();
      showSuccessToast('Change plan closed successfully');

      if (onSuccess) {
        onSuccess();
      }
    } else {
      setError('Failed to close change plan');
    }
  };

  // Handle execute
  const handleExecute = async () => {
    if (!canExecute) {
      setError('You do not have permission to execute change plans.');
      return;
    }

    if (changePlan.status !== 'Approved') {
      setError('Only Approved change plans can be executed.');
      return;
    }

    setError('');
    setExecutionStatus('executing');

    const success = await executeChangePlan(changePlan.id);

    if (success) {
      setExecutionStatus('success');
      // Ghost runs in the background (202 Accepted); direct executions
      // complete synchronously.
      if (changePlan.execution_backend === 'ghost') {
        showSuccessToast('Online Migrator execution started — progress will update below');
      } else {
        showSuccessToast('Change plan executed successfully');
      }

      // Wait a moment to show success state, then close and refresh
      setTimeout(() => {
        handleDialogClose();

        if (onSuccess) {
          onSuccess();
        }
      }, 1500);
    } else {
      setExecutionStatus('idle');
      setError('Failed to execute change plan');
    }
  };

  // Get action config
  const getActionConfig = (action: ActionType) => {
    switch (action) {
      case 'submit':
        return {
          label: 'Ready for Review',
          icon: Send,
          variant: 'default' as const,
          handler: handleSubmit,
        };
      case 'approve':
        return {
          label: 'Approve',
          icon: CheckCircle2,
          variant: 'default' as const,
          handler: handleApprove,
        };
      case 'retry':
        return {
          label: 'Retry',
          icon: CheckCircle2,
          variant: 'default' as const,
          handler: handleApprove,
        };
      case 'close':
        return {
          label: 'Close',
          icon: XCircle,
          variant: 'outline' as const,
          handler: handleClose,
        };
      case 'execute':
        return {
          label: 'Execute',
          icon: Play,
          variant: 'default' as const,
          handler: handleExecute,
        };
      case 'approve-cutover':
        return {
          label: 'Approve Cutover',
          icon: Play,
          variant: 'default' as const,
          handler: handleApproveCutover,
        };
      case 'abort':
        return {
          label: 'Abort Migration',
          icon: XCircle,
          variant: 'destructive' as const,
          handler: handleAbort,
        };
      case 'ghost-rollback':
        return {
          label: 'Rollback',
          icon: XCircle,
          variant: 'destructive' as const,
          handler: handleGhostRollback,
        };
      case 'ghost-drop-backup':
        return {
          label: 'Drop Backup Table',
          icon: XCircle,
          variant: 'outline' as const,
          handler: handleGhostDropBackup,
        };
    }
  };

  // Get dialog content based on action
  const getDialogContent = () => {
    switch (activeAction) {
      case 'submit':
        return {
          title: 'Submit Change Plan for Review',
          description:
            'Submit this change plan for review. SQL Review will run automatically to check for policy violations, and then approvers will be able to review and approve it.',
          content: (
            <div className="rounded-md border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-950/20">
              <div className="flex items-center gap-2 text-blue-800 dark:text-blue-200">
                <AlertTriangle className="h-4 w-4" />
                <span className="text-sm font-medium">
                  After submission, SQL Review will automatically check your SQL against configured
                  policies.
                </span>
              </div>
            </div>
          ),
          confirmLabel: 'Submit for Review',
          confirmIcon: Send,
        };

      case 'approve':
      case 'retry':
        return {
          title: activeAction === 'retry' ? 'Retry Failed Change Plan' : 'Approve Change Plan',
          description:
            activeAction === 'retry'
              ? 'Approve this change plan to retry execution. The status will transition to Approved, allowing you to execute again.'
              : 'Approve this change plan to proceed with execution. You can add optional comments below.',
          content: (
            <div className="space-y-4">
              {changePlan.has_errors && (
                <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
                  <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
                    <AlertTriangle className="h-4 w-4" />
                    <span className="text-sm font-medium">
                      Cannot approve change plan with SQL Review errors
                    </span>
                  </div>
                </div>
              )}
              <div>
                <label htmlFor="approve-comments" className="text-sm font-medium">
                  Comments (optional)
                </label>
                <Textarea
                  id="approve-comments"
                  placeholder="Add any comments about this approval..."
                  value={comments}
                  onChange={(e) => setComments(e.target.value)}
                  className="mt-1"
                  rows={4}
                  disabled={isSubmitting || changePlan.has_errors}
                />
              </div>
            </div>
          ),
          confirmLabel: 'Approve',
          confirmIcon: CheckCircle2,
          isDisabled: changePlan.has_errors,
        };

      case 'close':
        return {
          title: 'Close Change Plan',
          description:
            'Close this change plan to reject it. You can add optional comments explaining why.',
          content: (
            <div>
              <label htmlFor="close-comments" className="text-sm font-medium">
                Comments (optional)
              </label>
              <Textarea
                id="close-comments"
                placeholder="Add any comments about why this change plan is being closed..."
                value={comments}
                onChange={(e) => setComments(e.target.value)}
                className="mt-1"
                rows={4}
                disabled={isSubmitting}
              />
            </div>
          ),
          confirmLabel: 'Close',
          confirmIcon: XCircle,
          confirmVariant: 'destructive' as const,
        };

      case 'execute':
        return {
          title: 'Execute Change Plan',
          description:
            'Execute this change plan against the target database. This action cannot be undone.',
          content: (
            <div className="space-y-4">
              {changePlan.has_errors && (
                <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
                  <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
                    <AlertTriangle className="h-4 w-4" />
                    <span className="text-sm font-medium">
                      Cannot execute change plan with SQL Review errors
                    </span>
                  </div>
                </div>
              )}

              {executionStatus === 'executing' && (
                <div className="rounded-md border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950/20">
                  <div className="flex items-center gap-3">
                    <Loader2 className="h-5 w-5 animate-spin text-blue-600 dark:text-blue-400" />
                    <div>
                      <p className="text-sm font-medium text-blue-800 dark:text-blue-200">
                        Executing change plan...
                      </p>
                      <p className="text-xs text-blue-700 dark:text-blue-300 mt-1">
                        Please wait while the SQL statements are executed against the database.
                      </p>
                    </div>
                  </div>
                </div>
              )}

              {executionStatus === 'success' && (
                <div className="rounded-md border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-950/20">
                  <div className="flex items-center gap-3">
                    <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
                    <div>
                      <p className="text-sm font-medium text-green-800 dark:text-green-200">
                        Execution completed successfully!
                      </p>
                      <p className="text-xs text-green-700 dark:text-green-300 mt-1">
                        The change plan has been executed and the status has been updated.
                      </p>
                    </div>
                  </div>
                </div>
              )}

              {executionStatus === 'idle' && (
                <div className="rounded-md border p-4 space-y-2">
                  <p className="text-sm font-medium">Execution Details</p>
                  <ul className="text-sm text-muted-foreground space-y-1 list-disc list-inside">
                    <li>SQL statements will be executed in a transaction</li>
                    <li>Schema snapshots will be captured before and after execution</li>
                    <li>On success, the change plan status will be updated to Completed</li>
                    <li>
                      On failure, the transaction will be rolled back and status will be Failed
                    </li>
                  </ul>
                </div>
              )}
            </div>
          ),
          confirmLabel:
            executionStatus === 'success'
              ? 'Executed'
              : executionStatus === 'executing'
                ? 'Executing...'
                : 'Execute',
          confirmIcon:
            executionStatus === 'success'
              ? CheckCircle2
              : executionStatus === 'executing'
                ? Loader2
                : Play,
          isDisabled: changePlan.has_errors || executionStatus === 'success',
        };

      case 'approve-cutover':
        return {
          title: 'Approve Cutover',
          description:
            'The shadow table copy is complete and the migration is parked awaiting cutover. Approving performs the atomic table rename (under a second). This cannot be undone.',
          content: (
            <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950/20">
              <div className="flex items-center gap-2 text-amber-800 dark:text-amber-200">
                <AlertTriangle className="h-4 w-4" />
                <span className="text-sm font-medium">
                  The atomic rename swaps the new table into place immediately.
                </span>
              </div>
            </div>
          ),
          confirmLabel: 'Approve Cutover',
          confirmIcon: Play,
        };

      case 'abort':
        return {
          title: 'Abort Migration',
          description:
            'Abort the running migration before the cutover. The original table stays untouched; the shadow table is dropped.',
          content: (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
              <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
                <AlertTriangle className="h-4 w-4" />
                <span className="text-sm font-medium">
                  This stops the migration; it must be re-executed to try again.
                </span>
              </div>
            </div>
          ),
          confirmLabel: 'Abort Migration',
          confirmIcon: XCircle,
          confirmVariant: 'destructive' as const,
        };

      case 'ghost-rollback':
        return {
          title: 'Roll Back Migration',
          description:
            'Atomically swap the backup table back into place, reverting the schema change.',
          content: (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
              <div className="flex items-start gap-2 text-red-800 dark:text-red-200">
                <AlertTriangle className="h-4 w-4 mt-0.5" />
                <div className="text-sm">
                  <span className="font-medium">Data loss warning:</span> writes made to the live
                  table since the cutover are NOT in the backup. They will be preserved in a
                  sidelined table (_table_new) for manual reconciliation, but the live table reverts
                  to its pre-migration contents.
                </div>
              </div>
            </div>
          ),
          confirmLabel: 'Roll Back',
          confirmIcon: XCircle,
          confirmVariant: 'destructive' as const,
        };

      case 'ghost-drop-backup':
        return {
          title: 'Drop Backup Table',
          description:
            'Permanently drop the backup table retained by the migration. Only do this after your validation window has passed — it removes the rollback safety net.',
          content: (
            <div className="space-y-2">
              <div className="rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950/20">
                <div className="flex items-center gap-2 text-amber-800 dark:text-amber-200">
                  <AlertTriangle className="h-4 w-4" />
                  <span className="text-sm font-medium">
                    After dropping, rollback via table swap is no longer possible.
                  </span>
                </div>
              </div>
              <ul className="list-disc pl-5 text-sm text-muted-foreground space-y-1">
                <li>
                  The backup holds a full copy of the pre-migration table — dropping it permanently
                  deletes that copy.
                </li>
                <li>
                  Retained backups consume disk space roughly equal to the live table, so leaving
                  them behind grows storage costs.
                </li>
                <li>
                  gh-ost cannot start a new migration on the same table while an old backup (or
                  shadow table) is present — dropping it unblocks the next change on this table.
                </li>
              </ul>
            </div>
          ),
          confirmLabel: 'Drop Backup',
          confirmIcon: Trash2,
          confirmVariant: 'destructive' as const,
        };

      default:
        return null;
    }
  };

  const primaryAction = getPrimaryAction();
  const secondaryActions = getSecondaryActions();
  const dialogContent = activeAction ? getDialogContent() : null;

  // Don't render if no actions available
  if (!primaryAction && secondaryActions.length === 0) {
    return null;
  }

  return (
    <>
      <div className="flex items-center gap-2">
        {/* Primary Action Button */}
        {primaryAction &&
          (() => {
            const config = getActionConfig(primaryAction);
            const Icon = config.icon;
            // Ghost plans: submit stays visible but disabled until all
            // readiness checks pass (PR-checks semantics), with a tooltip
            // naming the failed check count.
            const ghostChecksFailed =
              primaryAction === 'submit' &&
              isGhostPlan &&
              planGhostChecks != null &&
              !planGhostChecks.all_passed;
            const failedCount = planGhostChecks
              ? planGhostChecks.checks.filter((c) => !c.passed).length
              : 0;
            const isDisabled =
              ((primaryAction === 'approve' || primaryAction === 'retry') &&
                changePlan.has_errors) ||
              (primaryAction === 'execute' && changePlan.has_errors) ||
              ghostChecksFailed;

            const button = (
              <Button
                variant={config.variant}
                onClick={() => setActiveAction(primaryAction)}
                disabled={isDisabled}
              >
                <Icon className="mr-2 h-4 w-4" />
                {config.label}
              </Button>
            );

            if (ghostChecksFailed) {
              return (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-block">{button}</span>
                  </TooltipTrigger>
                  <TooltipContent>
                    {failedCount} ghost readiness check{failedCount === 1 ? '' : 's'} failing —
                    resolve them before submitting for review.
                  </TooltipContent>
                </Tooltip>
              );
            }
            return button;
          })()}

        {/* Secondary Actions Dropdown */}
        {secondaryActions.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {secondaryActions.map((action) => {
                const config = getActionConfig(action);
                const Icon = config.icon;
                return (
                  <DropdownMenuItem key={action} onClick={() => setActiveAction(action)}>
                    <Icon className="mr-2 h-4 w-4" />
                    {config.label}
                  </DropdownMenuItem>
                );
              })}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      {/* Action Dialog */}
      {dialogContent && (
        <Dialog open={!!activeAction} onOpenChange={(open) => !open && handleDialogClose()}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{dialogContent.title}</DialogTitle>
              <DialogDescription>{dialogContent.description}</DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              {dialogContent.content}

              {error && (
                <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
                  <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
                    <AlertTriangle className="h-4 w-4" />
                    <span className="text-sm">{error}</span>
                  </div>
                </div>
              )}

              {ghostChecks && (
                <div className="rounded-md border p-3 space-y-2">
                  <p className="text-sm font-medium">Online Migrator readiness checks</p>
                  <ul className="space-y-1.5">
                    {ghostChecks.checks.map((check) => (
                      <li key={check.name} className="flex items-start gap-2 text-sm">
                        {check.passed ? (
                          <CheckCircle2 className="h-4 w-4 mt-0.5 text-green-600 dark:text-green-400 shrink-0" />
                        ) : (
                          <XCircle className="h-4 w-4 mt-0.5 text-red-600 dark:text-red-400 shrink-0" />
                        )}
                        <div>
                          <span className="font-medium">{check.name}</span>
                          <p className="text-muted-foreground text-xs">{check.detail}</p>
                        </div>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>

            <DialogFooter>
              <Button
                variant="outline"
                onClick={handleDialogClose}
                disabled={isSubmitting || executionStatus === 'success'}
              >
                {executionStatus === 'success' ? 'Close' : 'Cancel'}
              </Button>
              <Button
                variant={dialogContent.confirmVariant || 'default'}
                onClick={getActionConfig(activeAction!).handler}
                disabled={isSubmitting || dialogContent.isDisabled || executionStatus === 'success'}
              >
                {isSubmitting || executionStatus === 'executing' ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    {activeAction === 'execute' ? 'Executing...' : 'Processing...'}
                  </>
                ) : (
                  <>
                    {React.createElement(dialogContent.confirmIcon, { className: 'mr-2 h-4 w-4' })}
                    {dialogContent.confirmLabel}
                  </>
                )}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
};
