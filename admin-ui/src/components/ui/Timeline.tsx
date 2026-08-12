import { IconCircleFilled } from "@tabler/icons-react";
import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface TimelineItem {
  id: string;
  title: ReactNode;
  description?: ReactNode;
  timestamp?: ReactNode;
  icon?: IconElement;
  iconClassName?: string;
}

interface TimelineProps {
  items: TimelineItem[];
  className?: string;
}

const Timeline: FC<TimelineProps> = ({ items, className }) => {
  if (items.length === 0) {
    return null;
  }

  return (
    <div className={classNames("space-y-0", className)}>
      {items.map((item, index) => {
        const Icon = item.icon ?? IconCircleFilled;
        const isLast = index === items.length - 1;

        return (
          <div
            key={item.id}
            className="group relative flex gap-4 pb-6 last:pb-0"
          >
            <div className="relative flex flex-col items-center">
              <span
                className={classNames(
                  "relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2 border-white bg-gray-100 transition-colors dark:border-gray-950 dark:bg-gray-800",
                  item.iconClassName,
                )}
              >
                <Icon size={14} />
              </span>
              {!isLast && (
                <div className="absolute top-8 h-full w-px bg-gray-200 dark:bg-gray-800" />
              )}
            </div>

            <div className="min-w-0 flex-1 pt-1">
              <div className="flex items-baseline justify-between gap-2">
                <span className="text-sm font-medium text-gray-900 dark:text-gray-50">
                  {item.title}
                </span>
                {item.timestamp && (
                  <span className="shrink-0 text-xs text-gray-400 dark:text-gray-500">
                    {item.timestamp}
                  </span>
                )}
              </div>
              {item.description && (
                <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">
                  {item.description}
                </p>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default Timeline;
