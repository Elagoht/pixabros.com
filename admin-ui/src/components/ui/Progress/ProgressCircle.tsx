import classNames from "classnames";
import type { FC } from "react";

interface ProgressCircleProps {
  value: number;
  size?: number;
  strokeWidth?: number;
  showValue?: boolean;
  className?: string;
}

const ProgressCircle: FC<ProgressCircleProps> = ({
  value,
  size = 64,
  strokeWidth = 4,
  showValue = false,
  className,
}) => {
  const clamped = Math.min(100, Math.max(0, value));
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (clamped / 100) * circumference;

  return (
    <div
      className={classNames(
        "relative inline-flex items-center justify-center",
        className,
      )}
    >
      <svg width={size} height={size} className="-rotate-90">
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          className="stroke-gray-200 dark:stroke-gray-700"
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          className="stroke-primary-500 transition-all duration-300 ease-out"
        />
      </svg>
      {showValue && (
        <span className="absolute text-xs font-medium text-gray-700 dark:text-gray-300">
          {clamped}%
        </span>
      )}
    </div>
  );
};

export default ProgressCircle;
