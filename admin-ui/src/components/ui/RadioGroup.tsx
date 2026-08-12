import classNames from "classnames";
import { useField } from "formik";
import type { FC, ReactNode } from "react";

interface RadioOption {
  label: ReactNode;
  value: string;
}

interface RadioGroupProps {
  name: string;
  label?: string;
  options: RadioOption[];
  className?: string;
}

const RadioGroup: FC<RadioGroupProps> = ({
  name,
  label,
  options,
  className,
}) => {
  const [field, meta, helpers] = useField(name);
  const hasError = meta.touched && !!meta.error;

  return (
    <div className={classNames("w-full", className)}>
      {label && (
        <span className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </span>
      )}
      <div className="space-y-2">
        {options.map((opt) => {
          const checked = field.value === opt.value;
          return (
            <label
              key={opt.value}
              htmlFor={`${name}-${opt.value}`}
              className="relative flex cursor-pointer items-center gap-2"
            >
              <input
                id={`${name}-${opt.value}`}
                type="radio"
                name={name}
                value={opt.value}
                checked={checked}
                onChange={() => helpers.setValue(opt.value)}
                onBlur={field.onBlur}
                className={classNames(
                  "h-6 w-6 shrink-0 cursor-pointer appearance-none rounded-full border transition-all duration-200",
                  "border-gray-300 bg-gray-50 dark:border-gray-600 dark:bg-gray-800/50",
                  "checked:border-primary-500 dark:checked:border-primary-500",
                  "hover:shadow-md hover:shadow-primary-400/25 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950",
                  {
                    "border-red-500 hover:shadow-red-400/25 hover:ring-red-400/30":
                      hasError,
                  },
                )}
              />

              <span
                className={classNames(
                  "absolute left-1 top-1 h-4 w-4 rounded-full bg-gradient-to-b from-primary-400 to-primary-600 shadow-sm shadow-primary-500/50 transition-all",
                  {
                    "scale-0 opacity-0": !checked,
                    "scale-100 opacity-100": checked,
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

export default RadioGroup;
