import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface FieldSetProps {
  legend: string;
  icon?: IconElement;
  description?: string;
  error?: string;
  disabled?: boolean;
  className?: string;
  children: ReactNode;
}

const FieldSet: FC<FieldSetProps> = ({
  legend,
  icon: Icon,
  description,
  error,
  disabled,
  className,
  children,
}) => (
  <fieldset
    disabled={disabled}
    className={classNames(
      "rounded-lg border px-4 pb-4 pt-1 transition-colors duration-150",
      error
        ? "border-red-300 dark:border-red-800"
        : "border-gray-200 dark:border-gray-700",
      disabled && "opacity-50",
      className,
    )}
  >
    <legend className="flex items-center gap-1.5 px-2 text-sm font-semibold text-gray-900 dark:text-gray-50">
      {Icon && <Icon size={16} className="text-gray-500 dark:text-gray-400" />}
      {legend}
    </legend>
    {description && (
      <p className="mb-3 px-2 text-xs text-gray-500 dark:text-gray-400">
        {description}
      </p>
    )}
    <div className="space-y-3">{children}</div>
    {error && (
      <p className="mt-2 px-2 text-xs text-red-500 dark:text-red-400">
        {error}
      </p>
    )}
  </fieldset>
);

export default FieldSet;
