import classNames from "classnames";
import type { FC } from "react";

interface ProgressBarProps {
  value: number;
  size?: "sm" | "md" | "lg";
  showValue?: boolean;
  className?: string;
}

const sizeClasses = { sm: "h-1.5", md: "h-2.5", lg: "h-4" };

const ProgressBar: FC<ProgressBarProps> = ({
  value,
  size = "md",
  showValue = false,
  className,
}) => {
  const clamped = Math.min(100, Math.max(0, value));

  return (
    <div className={classNames("w-full", className)}>
      {showValue && (
        <div className="mb-1 flex justify-between text-xs text-gray-500 dark:text-gray-400">
          <span />
          <span>{clamped}%</span>
        </div>
      )}
      <div
        className={classNames(
          "w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700",
          sizeClasses[size],
        )}
      >
        <div
          className="h-full rounded-full bg-primary-500 transition-all duration-300 ease-out"
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
};

export default ProgressBar;
