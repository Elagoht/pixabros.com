import { IconAlertTriangle, IconCheck } from "@tabler/icons-react";
import classNames from "classnames";
import type { FC } from "react";

interface StepperProps {
  steps: string[];
  activeStep: number;
  onStepClick?: (step: number) => void;
  errorSteps?: Set<number>;
  className?: string;
}

const Stepper: FC<StepperProps> = ({
  steps,
  activeStep,
  onStepClick,
  errorSteps,
  className,
}) => (
  <div className={classNames("flex items-start", className)}>
    {steps.map((label, i) => {
      const isCompleted = i < activeStep;
      const isActive = i === activeStep;
      const hasError = errorSteps?.has(i);

      return (
        <div key={i} className="flex flex-1 flex-col items-center">
          <div className="flex w-full items-center justify-center">
            <div
              className={classNames(
                "h-1 flex-1 rounded-full transition-all duration-300",
                i === 0 && "invisible",
                i > 0 &&
                  (i - 1 < activeStep
                    ? "bg-primary-500"
                    : "bg-gray-200 dark:bg-gray-700"),
              )}
            />
            <button
              type="button"
              disabled={!onStepClick}
              onClick={() => onStepClick?.(i)}
              className={classNames(
                "flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm font-medium transition-all duration-200",
                hasError &&
                  "bg-red-100 text-red-600 ring-2 ring-red-300 dark:bg-red-900/30 dark:text-red-400 dark:ring-red-800",
                !hasError && isCompleted && "bg-primary-500 text-white",
                !hasError &&
                  isActive &&
                  "bg-primary-500 text-white ring-4 ring-primary-100/50 dark:ring-primary-900/30",
                !(hasError || isCompleted || isActive) &&
                  "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700",
                onStepClick && "cursor-pointer hover:scale-105",
              )}
            >
              {hasError ? (
                <IconAlertTriangle size={18} strokeWidth={3} />
              ) : isCompleted ? (
                <IconCheck size={18} strokeWidth={3} />
              ) : (
                i + 1
              )}
            </button>
            <div
              className={classNames(
                "h-1 flex-1 rounded-full transition-all duration-300",
                i === steps.length - 1 && "invisible",
                i < steps.length - 1 &&
                  (i < activeStep
                    ? "bg-primary-500"
                    : "bg-gray-200 dark:bg-gray-700"),
              )}
            />
          </div>
          <span
            className={classNames(
              "mt-2 text-xs font-medium transition-colors duration-200",
              hasError && "text-red-600 dark:text-red-400",
              !hasError && isActive && "text-primary-700 dark:text-primary-300",
              !hasError && isCompleted && "text-gray-700 dark:text-gray-300",
              !(hasError || isCompleted || isActive) &&
                "text-gray-400 dark:text-gray-500",
            )}
          >
            {label}
          </span>
        </div>
      );
    })}
  </div>
);

export default Stepper;
