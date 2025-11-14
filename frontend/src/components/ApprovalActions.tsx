import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { changePlanService } from '@/lib/api/services/changePlans';
import type { ChangePlan } from '@/lib/api/types';
import { CheckCircle2, XCircle, Loader2, AlertTriangle } from 'lucide-react';
import { showSuccessToast, handleApiError } from '@/lib/errorHandling';

interface ApprovalActionsProps {
  changePlan: ChangePlan;
  onSuccess?: () => void;
  canApprove: boolean;
  canReject: boolean;
}

export const ApprovalActions: React.FC<ApprovalActionsProps> = ({
  changePlan,
  onSuccess,
  canApprove,
  canReject,
}) => {
  const [isApproveDialogOpen, setIsApproveDialogOpen] = useState(false);
  const [isCloseDialogOpen, setIsCloseDialogOpen] = useState(false);
  const [comments, setComments] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');

  // Only show actions if change plan is in Open status or Failed status (can retry)
  const canPerformActions = changePlan.status === 'Open' || changePlan.status === 'Failed';

  // Reset state when dialog closes
  const handleDialogClose = () => {
    setComments('');
    setError('');
  };

  const handleApprove = async () => {
    if (!canApprove) {
      setError('You do not have permission to approve change plans.');
      return;
    }

    try {
      setIsSubmitting(true);
      setError('');

      await changePlanService.approveChangePlan(changePlan.id, {
        comments: comments.trim() || undefined,
      });

      setIsApproveDialogOpen(false);
      handleDialogClose();
      showSuccessToast(
        changePlan.status === 'Failed'
          ? 'Change plan approved for retry. You can now execute it again.'
          : 'Change plan approved successfully',
      );

      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      const errorMsg = handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to approve change plan',
      });
      setError(errorMsg);
      console.error('Failed to approve change plan:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = async () => {
    if (!canReject) {
      setError('You do not have permission to close change plans.');
      return;
    }

    try {
      setIsSubmitting(true);
      setError('');

      await changePlanService.closeChangePlan(changePlan.id, {
        comments: comments.trim() || undefined,
      });

      setIsCloseDialogOpen(false);
      handleDialogClose();
      showSuccessToast('Change plan closed successfully');

      if (onSuccess) {
        onSuccess();
      }
    } catch (err) {
      const errorMsg = handleApiError(err, {
        showToast: true,
        toastTitle: 'Failed to close change plan',
      });
      setError(errorMsg);
      console.error('Failed to close change plan:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Don't render if user can't approve/reject or change plan is not in Open/Failed status
  if ((!canApprove && !canReject) || !canPerformActions) {
    return null;
  }

  return (
    <div className="flex items-center gap-2">
      {canApprove && (
        <Dialog open={isApproveDialogOpen} onOpenChange={setIsApproveDialogOpen}>
          <DialogTrigger asChild>
            <Button
              variant="default"
              onClick={() => {
                setIsApproveDialogOpen(true);
                handleDialogClose();
              }}
              disabled={changePlan.has_errors}
            >
              <CheckCircle2 className="mr-2 h-4 w-4" />
              {changePlan.status === 'Failed' ? 'Retry' : 'Approve'}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>
                {changePlan.status === 'Failed'
                  ? 'Retry Failed Change Plan'
                  : 'Approve Change Plan'}
              </DialogTitle>
              <DialogDescription>
                {changePlan.status === 'Failed'
                  ? 'Approve this change plan to retry execution. The status will transition to Approved, allowing you to execute again.'
                  : 'Approve this change plan to proceed with execution. You can add optional comments below.'}
              </DialogDescription>
            </DialogHeader>

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

            <div className="space-y-4">
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
                onClick={() => {
                  setIsApproveDialogOpen(false);
                  handleDialogClose();
                }}
                disabled={isSubmitting}
              >
                Cancel
              </Button>
              <Button onClick={handleApprove} disabled={isSubmitting || changePlan.has_errors}>
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Approving...
                  </>
                ) : (
                  <>
                    <CheckCircle2 className="mr-2 h-4 w-4" />
                    Approve
                  </>
                )}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      {canReject && (
        <Dialog open={isCloseDialogOpen} onOpenChange={setIsCloseDialogOpen}>
          <DialogTrigger asChild>
            <Button
              variant="outline"
              onClick={() => {
                setIsCloseDialogOpen(true);
                handleDialogClose();
              }}
            >
              <XCircle className="mr-2 h-4 w-4" />
              Close
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Close Change Plan</DialogTitle>
              <DialogDescription>
                Close this change plan to reject it. You can add optional comments explaining why.
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
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
                onClick={() => {
                  setIsCloseDialogOpen(false);
                  handleDialogClose();
                }}
                disabled={isSubmitting}
              >
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleClose} disabled={isSubmitting}>
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Closing...
                  </>
                ) : (
                  <>
                    <XCircle className="mr-2 h-4 w-4" />
                    Close
                  </>
                )}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
};
