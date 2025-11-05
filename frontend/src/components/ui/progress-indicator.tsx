import type React from 'react';
import { cn } from '@/lib/utils';
import { Progress } from '@/components/ui/progress';

interface ProgressIndicatorProps {
  value?: number;
  max?: number;
  className?: string;
  showLabel?: boolean;
  label?: string;
  indeterminate?: boolean;
}

export function ProgressIndicator({
  value,
  max = 100,
  className,
  showLabel = false,
  label,
  indeterminate = false,
}: ProgressIndicatorProps) {
  const percentage = indeterminate ? undefined : value !== undefined ? (value / max) * 100 : 0;

  return (
    <div className={cn('space-y-2', className)}>
      {(showLabel || label) && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted-foreground">{label || 'Progress'}</span>
          {!indeterminate && value !== undefined && (
            <span className="text-muted-foreground">{Math.round(percentage || 0)}%</span>
          )}
        </div>
      )}
      <Progress value={indeterminate ? undefined : percentage} className="h-2" />
    </div>
  );
}
