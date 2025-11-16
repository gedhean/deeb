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
import { useExecuteChangePlan } from '@/hooks/api/useChangePlansSWR';
import type { ChangePlan } from '@/lib/api/types';
import { Play, Loader2, AlertTriangle, CheckCircle2 } from 'lucide-react';
import { showSuccessToast } from '@/lib/errorHandling';

interface ExecuteButtonProps {
  changePlan: ChangePlan;
  onSuccess?: () => void;
  canExecute: boolean;
}

export const ExecuteButton: React.FC<ExecuteButtonProps> = ({
  changePlan,
  onSuccess,
  canExecute,
}) => {
  // Use SWR mutation hook for consistent cache management
  const { executeChangePlan, isLoading: isExecuting } = useExecuteChangePlan();

  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [error, setError] = useState('');
  const [executionStatus, setExecutionStatus] = useState<'idle' | 'executing' | 'success'>('idle');

  // Only enable for Approved status
  const canExecuteThisPlan = changePlan.status === 'Approved';

  // Reset state when dialog closes
  const handleDialogClose = () => {
    setError('');
    setExecutionStatus('idle');
  };

  const handleExecute = async () => {
    if (!canExecute) {
      setError('You do not have permission to execute change plans.');
      return;
    }

    if (!canExecuteThisPlan) {
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
        setIsDialogOpen(false);
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

  // Don't render if user can't execute
  if (!canExecute) {
    return null;
  }

  return (
    <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
      <DialogTrigger asChild>
        <Button
          variant="default"
          onClick={() => {
            setIsDialogOpen(true);
            handleDialogClose();
          }}
          disabled={!canExecuteThisPlan || changePlan.has_errors}
        >
          <Play className="mr-2 h-4 w-4" />
          Execute
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Execute Change Plan</DialogTitle>
          <DialogDescription>
            Execute this change plan against the target database. This action cannot be undone.
          </DialogDescription>
        </DialogHeader>

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

        {!canExecuteThisPlan && (
          <div className="rounded-md border border-yellow-200 bg-yellow-50 p-3 dark:border-yellow-800 dark:bg-yellow-950/20">
            <div className="flex items-center gap-2 text-yellow-800 dark:text-yellow-200">
              <AlertTriangle className="h-4 w-4" />
              <span className="text-sm">
                Only Approved change plans can be executed. Current status: {changePlan.status}
              </span>
            </div>
          </div>
        )}

        <div className="space-y-4">
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

          {error && (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-800 dark:bg-red-950/20">
              <div className="flex items-center gap-2 text-red-800 dark:text-red-200">
                <AlertTriangle className="h-4 w-4" />
                <span className="text-sm">{error}</span>
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
                <li>On failure, the transaction will be rolled back and status will be Failed</li>
              </ul>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              setIsDialogOpen(false);
              handleDialogClose();
            }}
            disabled={isExecuting || executionStatus === 'success'}
          >
            {executionStatus === 'success' ? 'Close' : 'Cancel'}
          </Button>
          <Button
            onClick={handleExecute}
            disabled={
              isExecuting ||
              !canExecuteThisPlan ||
              changePlan.has_errors ||
              executionStatus === 'success'
            }
          >
            {isExecuting ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Executing...
              </>
            ) : executionStatus === 'success' ? (
              <>
                <CheckCircle2 className="mr-2 h-4 w-4" />
                Executed
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                Execute
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
