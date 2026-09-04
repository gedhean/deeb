import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
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
import type { ChangePlan } from '@/lib/api/types';
import {
  Send,
  CheckCircle2,
  XCircle,
  Play,
  Loader2,
  AlertTriangle,
  MoreVertical,
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
  | 'abort';

export const ChangePlanActions: React.FC<ChangePlanActionsProps> = ({
  changePlan,
  onSuccess,
  canApprove,
  canReject,
  canExecute,
}) => {
  // Use SWR mutation hooks for consistent cache management
  const { updateChangePlan, isLoading: isUpdating, error: updateError } = useUpdateChangePlanSWR();
  const { approveChangePlan, isLoading: isApproving } = useApproveChangePlan();
  const { closeChangePlan, isLoading: isClosing } = useCloseChangePlan();
  const { executeChangePlan, isLoading: isExecuting } = useExecuteChangePlan();

  const [activeAction, setActiveAction] = useState<ActionType | null>(null);
  const [comments, setComments] = useState('');
  const [error, setError] = useState('');
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

  const isGhostPlan = changePlan.execution_backend === 'ghost';

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
      case 'Failed':
        if (canReject) {
          actions.push('close');
        }
        break;
    }

    return actions;
  };

  // Handle submit for review
  const handleSubmit = async () => {
    setError('');

    const result = await updateChangePlan(changePlan.id, {
      status: 'Open',
    });

    if (result) {
      handleDialogClose();
      showSuccessToast('Change plan submitted for review. SQL Review will run automatically.');

      if (onSuccess) {
        onSuccess();
      }
    } else {
      setError(`Failed to submit change plan for review: ${updateError || 'Unknown error'}`);
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
      showSuccessToast('Change plan executed successfully');

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
            const isDisabled =
              ((primaryAction === 'approve' || primaryAction === 'retry') &&
                changePlan.has_errors) ||
              (primaryAction === 'execute' && changePlan.has_errors);

            return (
              <Button
                variant={config.variant}
                onClick={() => setActiveAction(primaryAction)}
                disabled={isDisabled}
              >
                <Icon className="mr-2 h-4 w-4" />
                {config.label}
              </Button>
            );
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
