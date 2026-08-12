import { IconCheck } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import type { FC, ReactNode } from "react";

interface CheckboxOption {
  label: ReactNode;
  value: string;
}

interface CheckboxGroupProps {
  name: string;
  label?: string;
  options: CheckboxOption[];
  className?: string;
}

const CheckboxGroup: FC<CheckboxGroupProps> = ({
  name,
  label,
  options,
  className,
}) => {
  const [field, meta, helpers] = useField<string[]>(name);
  const hasError = meta.touched && !!meta.error;
  const selected = field.value ?? [];

  const handleChange = (value: string) => {
    const next = selected.includes(value)
      ? selected.filter((v: string) => v !== value)
      : [...selected, value];
    helpers.setValue(next);
  };

  return (
    <div className={classNames("w-full", className)}>
      {label && (
        <span className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </span>
      )}
      <div className="space-y-2">
        {options.map((opt) => {
          const checked = selected.includes(opt.value);
          return (
            <label
              key={opt.value}
              htmlFor={`${name}-${opt.value}`}
              className="relative flex cursor-pointer items-center gap-2"
            >
              <input
                id={`${name}-${opt.value}`}
                type="checkbox"
                checked={checked}
                onChange={() => handleChange(opt.value)}
                onBlur={field.onBlur}
                className={classNames(
                  "h-6 w-6 shrink-0 cursor-pointer appearance-none rounded-md border transition-all duration-200 ease-out",
                  "border-gray-300 bg-white dark:border-gray-600 dark:bg-gray-900",
                  "checked:border-primary-500 checked:bg-primary-500 dark:checked:border-primary-500 dark:checked:bg-primary-500",
                  "hover:shadow-md hover:shadow-primary-400/25 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950",
                  {
                    "border-red-500 hover:shadow-red-400/25 hover:ring-red-400/30":
                      hasError,
                  },
                )}
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
              <span className="select-none text-sm text-gray-700 dark:text-gray-300">
                {opt.label}
              </span>
            </label>
          );
        })}
      </div>
      {hasError && (
        <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
          {meta.error}
        </p>
      )}
    </div>
  );
};

export default CheckboxGroup;
