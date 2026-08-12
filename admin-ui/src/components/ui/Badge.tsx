import classNames from "classnames";
import type { FC, ReactNode } from "react";

type BadgeVariant =
  | "default"
  | "secondary"
  | "success"
  | "warning"
  | "destructive"
  | "outline";

interface BadgeProps {
  variant?: BadgeVariant;
  children: ReactNode;
  className?: string;
}

const variantClasses: Record<BadgeVariant, string> = {
  default: "bg-primary-500 text-white dark:bg-primary-600",
  secondary: "bg-secondary-700 text-white dark:bg-secondary-800",
  success: "bg-green-500 text-white dark:bg-green-600",
  warning: "bg-yellow-500 text-white dark:bg-yellow-600",
  destructive: "bg-red-500 text-white dark:bg-red-600",
  outline:
    "border border-primary-300 bg-white/90 text-primary-700 dark:border-primary-700 dark:bg-gray-900/80 dark:text-primary-300",
};

const Badge: FC<BadgeProps> = ({
  variant = "default",
  children,
  className,
}) => {
  return (
    <span
      className={classNames(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-all duration-200",
        variantClasses[variant],
        className,
      )}
    >
      {children}
    </span>
  );
};

export default Badge;
