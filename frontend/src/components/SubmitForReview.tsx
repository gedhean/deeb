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
import { useUpdateChangePlanSWR } from '@/hooks/api/useChangePlansSWR';
import type { ChangePlan } from '@/lib/api/types';
import { Send, Loader2, AlertTriangle } from 'lucide-react';
import { showSuccessToast } from '@/lib/errorHandling';

interface SubmitForReviewProps {
  changePlan: ChangePlan;
  onSuccess?: () => void;
}

export const SubmitForReview: React.FC<SubmitForReviewProps> = ({ changePlan, onSuccess }) => {
  // Use SWR mutation hook for consistent cache management
  const {
    updateChangePlan,
    isLoading: isSubmitting,
    error: updateError,
  } = useUpdateChangePlanSWR();

  const [isDialogOpen, setIsDialogOpen] = useState(false);

  // Only show if change plan is in Draft status
  const canSubmit = changePlan.status === 'Draft';

  // Check if user is the author (for now, we'll let backend handle authorization)
  // Backend will check if user is the author and has Project Developer role

  const handleSubmit = async () => {
    // Update status from Draft to Open
    const result = await updateChangePlan(changePlan.id, {
      status: 'Open',
    });

    if (result) {
      setIsDialogOpen(false);
      showSuccessToast('Change plan submitted for review. SQL Review will run automatically.');

      if (onSuccess) {
        onSuccess();
      }
    }
    // Error is handled by the hook and will be available via updateError
  };

  // Don't render if change plan is not in Draft status
  if (!canSubmit) {
    return null;
  }

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>
        <Button variant="default" onClick={() => setIsDialogOpen(true)}>
          <Send className="mr-2 h-4 w-4" />
          Submit for Review
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Submit Change Plan for Review</DialogTitle>
          <DialogDescription>
            Submit this change plan for review. SQL Review will run automatically to check for
            policy violations, and then approvers will be able to review and approve it.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-md border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-950/20">
            <div className="flex items-center gap-2 text-blue-800 dark:text-blue-200">
              <AlertTriangle className="h-4 w-4" />
              <span className="text-sm font-medium">
                After submission, SQL Review will automatically check your SQL against configured
                policies.
              </span>
            </div>
          </div>

          {updateError && (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
              <div className="flex items-start gap-2 text-red-800 dark:text-red-200">
                <AlertTriangle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                <span className="text-sm">{updateError}</span>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setIsDialogOpen(false)} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Submitting...
              </>
            ) : (
              <>
                <Send className="mr-2 h-4 w-4" />
                Submit for Review
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
