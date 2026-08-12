import { IconInbox } from "@tabler/icons-react";
import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface EmptyStateProps {
  icon?: IconElement;
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}

const EmptyState: FC<EmptyStateProps> = ({
  icon,
  title,
  description,
  action,
  className,
}) => {
  const Icon = icon ?? IconInbox;

  return (
    <div
      className={classNames(
        "flex flex-col items-center justify-center rounded-xl border border-dashed border-gray-200/60 bg-gray-50/30 py-12 text-center shadow-sm shadow-gray-500/10 dark:border-gray-700/60 dark:bg-gray-900/30",
        className,
      )}
    >
      <Icon
        size={40}
        className="mb-3 text-gray-300 transition-transform duration-300 hover:scale-110 dark:text-gray-600"
        strokeWidth={1.5}
      />
      <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {title}
      </p>
      {description && (
        <p className="mt-1 text-sm text-gray-400 dark:text-gray-500">
          {description}
        </p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
};

export default EmptyState;
