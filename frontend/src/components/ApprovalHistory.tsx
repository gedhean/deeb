import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { changePlanService } from '@/lib/api/services/changePlans';
import { usersService } from '@/lib/api/services/users';
import { APIError } from '@/lib/api/types';
import type { Approval, User } from '@/lib/api/types';
import { CheckCircle2, XCircle, User as UserIcon, Calendar, MessageSquare } from 'lucide-react';

interface ApprovalHistoryProps {
  changePlanId: string;
}

export const ApprovalHistory: React.FC<ApprovalHistoryProps> = ({ changePlanId }) => {
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [approvers, setApprovers] = useState<Map<string, User>>(new Map());
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadApprovalHistory();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [changePlanId]);

  const loadApprovalHistory = async () => {
    try {
      setIsLoading(true);
      setError('');

      // Load approval history
      const history = await changePlanService.getApprovalHistory(changePlanId);
      setApprovals(history.approvals || []);

      // Load approver information
      if (history.approvals && history.approvals.length > 0) {
        const approverIds = [...new Set(history.approvals.map((a) => a.approver_id))];
        const usersRes = await usersService.listUsers({ limit: 100 });

        const approversMap = new Map<string, User>();
        approverIds.forEach((approverId) => {
          const approver = usersRes.items.find((u) => u.id === approverId);
          if (approver) {
            approversMap.set(approverId, approver);
          }
        });

        setApprovers(approversMap);
      }
    } catch (err) {
      if (err instanceof APIError) {
        setError(err.message || 'Failed to load approval history.');
      } else {
        setError('Failed to load approval history. Please try again.');
      }
      console.error('Failed to load approval history:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const formatDate = (dateString: string): string => {
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        month: 'long',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return dateString;
    }
  };

  const getActionLabel = (status: 'Approved' | 'Rejected'): string => {
    return status === 'Approved' ? 'Approved' : 'Closed';
  };

  const getActionIcon = (status: 'Approved' | 'Rejected') => {
    return status === 'Approved' ? (
      <CheckCircle2 className="h-4 w-4" />
    ) : (
      <XCircle className="h-4 w-4" />
    );
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Approval History</CardTitle>
          <CardDescription>
            History of approvals and rejections for this change plan
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-20 w-full" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Approval History</CardTitle>
          <CardDescription>
            History of approvals and rejections for this change plan
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">{error}</p>
        </CardContent>
      </Card>
    );
  }

  if (approvals.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Approval History</CardTitle>
          <CardDescription>
            History of approvals and rejections for this change plan
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">No approval history available yet.</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Approval History</CardTitle>
        <CardDescription>History of approvals and rejections for this change plan</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {approvals.map((approval) => {
            const approver = approvers.get(approval.approver_id);
            const isApproved = approval.status === 'Approved';

            return (
              <div key={approval.id} className="rounded-lg border p-4 space-y-3">
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-3 flex-1">
                    <div
                      className={`mt-1 ${
                        isApproved
                          ? 'text-green-600 dark:text-green-400'
                          : 'text-red-600 dark:text-red-400'
                      }`}
                    >
                      {getActionIcon(approval.status)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <Badge
                          variant={isApproved ? 'default' : 'destructive'}
                          className={
                            isApproved
                              ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                              : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                          }
                        >
                          {getActionLabel(approval.status)}
                        </Badge>
                      </div>

                      <div className="space-y-2 mt-3">
                        <div className="flex items-center gap-2 text-sm">
                          <UserIcon className="h-4 w-4 text-muted-foreground" />
                          <span className="text-muted-foreground">Author:</span>
                          <span className="font-medium">
                            {approver?.email || approval.approver_id}
                          </span>
                        </div>

                        <div className="flex items-center gap-2 text-sm">
                          <Calendar className="h-4 w-4 text-muted-foreground" />
                          <span className="text-muted-foreground">Date:</span>
                          <span className="font-medium">
                            {formatDate(approval.approved_at || approval.created_at)}
                          </span>
                        </div>

                        {approval.comments && (
                          <div className="flex items-start gap-2 text-sm mt-2">
                            <MessageSquare className="h-4 w-4 text-muted-foreground mt-0.5" />
                            <div className="flex-1">
                              <span className="text-muted-foreground">Comment:</span>
                              <p className="mt-1 text-muted-foreground whitespace-pre-wrap">
                                {approval.comments}
                              </p>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
};
