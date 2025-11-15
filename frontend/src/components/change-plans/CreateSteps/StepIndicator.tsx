import React from 'react';
import { Check } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface StepIndicatorProps {
  currentStep: number;
  totalSteps: number;
  steps: {
    label: string;
    description: string;
  }[];
}

export const StepIndicator: React.FC<StepIndicatorProps> = ({ currentStep, totalSteps, steps }) => {
  return (
    <nav aria-label="Progress">
      <ol role="list" className="flex items-center">
        {steps.map((step, index) => {
          const stepNumber = index + 1;
          const isCompleted = stepNumber < currentStep;
          const isCurrent = stepNumber === currentStep;
          const isUpcoming = stepNumber > currentStep;

          return (
            <li
              key={step.label}
              className={cn('relative', index !== totalSteps - 1 && 'pr-8 sm:pr-20 flex-1')}
            >
              {/* Connector line (shown for all except last step) */}
              {index !== totalSteps - 1 && (
                <div
                  className="absolute inset-0 flex items-center"
                  aria-hidden="true"
                  style={{ left: '2rem' }}
                >
                  <div
                    className={cn(
                      'h-0.5 w-full mt-1',
                      isCompleted ? 'bg-primary' : 'bg-muted-foreground/30',
                    )}
                  />
                </div>
              )}

              {/* Step content */}
              <div className="relative flex items-center group">
                {/* Step circle */}
                <span
                  className={cn(
                    'h-8 w-8 rounded-full flex items-center justify-center border-2 relative z-10 transition-colors',
                    isCompleted &&
                      'bg-primary border-primary text-primary-foreground hover:bg-primary/90',
                    isCurrent && 'border-primary bg-background text-primary ring-4 ring-primary/20',
                    isUpcoming && 'border-muted-foreground/30 bg-background text-muted-foreground',
                  )}
                  aria-current={isCurrent ? 'step' : undefined}
                >
                  {isCompleted ? (
                    <Check className="h-4 w-4" />
                  ) : (
                    <span className="text-sm font-semibold">{stepNumber}</span>
                  )}
                </span>

                {/* Step text */}
                <div className="ml-4 min-w-0 flex flex-col">
                  <span
                    className={cn(
                      'text-sm font-medium pb-1',
                      isCurrent && 'text-primary',
                      isCompleted && 'text-foreground',
                      isUpcoming && 'text-muted-foreground',
                    )}
                  >
                    {step.label}
                  </span>
                  <span className="text-xs text-muted-foreground hidden sm:block">
                    {step.description}
                  </span>
                </div>
              </div>
            </li>
          );
        })}
      </ol>
    </nav>
  );
};
