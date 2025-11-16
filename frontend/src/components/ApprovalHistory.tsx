import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { useUsers } from '@/hooks/api/useUsersSWR';
import { useApprovalsSWR } from '@/hooks/api/useApprovalsSWR';
import { CheckCircle2, XCircle, User as UserIcon, Calendar, MessageSquare } from 'lucide-react';

interface ApprovalHistoryProps {
  changePlanId: string;
}

export const ApprovalHistory: React.FC<ApprovalHistoryProps> = ({ changePlanId }) => {
  // Use SWR hook for approvals with automatic request deduplication
  const { approvals, isLoading, error } = useApprovalsSWR(changePlanId);

  // Use cached users from store
  const { getUserById } = useUsers();

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
            const approver = getUserById(approval.approver_id);
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
