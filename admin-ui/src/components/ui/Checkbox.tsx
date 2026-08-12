import { IconCheck } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import type { FC } from "react";

interface CheckboxProps {
  name: string;
  label?: string;
  value?: string;
  className?: string;
}

const Checkbox: FC<CheckboxProps> = ({ name, label, value, className }) => {
  const [field, meta, helpers] = useField({ name, type: "checkbox", value });
  const hasError = meta.touched && !!meta.error;
  const id = value ? `${name}-${value}` : name;
  const checked = field.checked;

  return (
    <div className={classNames("relative flex items-center gap-2", className)}>
      <input
        id={id}
        type="checkbox"
        className={classNames(
          "h-6 w-6 shrink-0 cursor-pointer appearance-none rounded-md border transition-all duration-200 ease-out",
          "border-gray-300 bg-gray-50 dark:border-gray-600 dark:bg-gray-800/50",
          "checked:border-primary-500 checked:bg-primary-500 dark:checked:border-primary-500 dark:checked:bg-primary-500",
          "hover:shadow-md hover:shadow-primary-400/25 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950",
          {
            "border-red-500 hover:shadow-red-400/25 hover:ring-red-400/30":
              hasError,
          },
        )}
        checked={checked}
        onChange={() => helpers.setValue(!checked)}
        onBlur={field.onBlur}
      />
      <IconCheck
        size={20}
        strokeWidth={2}
        className={classNames(
          "pointer-events-none absolute top-0.5 text-white transition-all",
          {
            "-rotate-90 opacity-0": !checked,
            "translate-x-0.5": checked,
          },
        )}
      />
      {label && (
        <label
          htmlFor={id}
          className="flex-1 cursor-pointer select-none text-sm text-gray-700 dark:text-gray-300"
        >
          {label}
        </label>
      )}
      {hasError && !value && (
        <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
          {meta.error}
        </p>
      )}
    </div>
  );
};

export default Checkbox;
