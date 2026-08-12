import { IconPhone } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { type ComponentPropsWithoutRef, forwardRef, useCallback } from "react";

interface PhoneInputProps
  extends Omit<ComponentPropsWithoutRef<"input">, "name" | "type"> {
  name: string;
  label?: string;
}

const stripNonDigits = (val: string): string => val.replace(/[^+0-9]/g, "");

const formatTRPhone = (digits: string): string => {
  let num = digits.replace("+", "").slice(0, 12);
  if (num.startsWith("90")) {
    num = num.slice(2);
  } else if (num.startsWith("0")) {
    num = num.slice(1);
  }
  num = num.slice(0, 10);
  if (num.length === 0) {
    return "";
  }
  if (num.length <= 3) {
    return `+90 (${num}`;
  }
  if (num.length <= 6) {
    return `+90 (${num.slice(0, 3)}) ${num.slice(3)}`;
  }
  if (num.length <= 8) {
    return `+90 (${num.slice(0, 3)}) ${num.slice(3, 6)} ${num.slice(6)}`;
  }
  return `+90 (${num.slice(0, 3)}) ${num.slice(3, 6)} ${num.slice(6, 8)} ${num.slice(8)}`;
};

const PhoneInput = forwardRef<HTMLInputElement, PhoneInputProps>(
  ({ name, label, className, onChange, ...props }, ref) => {
    const [field, meta, helpers] = useField<string>(name);
    const hasError = meta.touched && !!meta.error;
    const raw = field.value ?? "";

    const handleChange = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        const digits = stripNonDigits(e.target.value).slice(0, 13);
        helpers.setValue(digits);
        onChange?.(e);
      },
      [helpers, onChange],
    );

    return (
      <div className="w-full">
        {label && (
          <label
            htmlFor={name}
            className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {label}
          </label>
        )}
        <div className="relative">
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-gray-400 dark:text-gray-500">
            <IconPhone size={16} />
          </div>
          <input
            ref={ref}
            id={name}
            type="tel"
            inputMode="tel"
            value={raw ? formatTRPhone(raw) : ""}
            onChange={handleChange}
            onBlur={field.onBlur}
            placeholder="+90 (5XX) XXX XX XX"
            className={classNames(
              "w-full rounded-lg border bg-gray-50 px-3 py-2 pl-9 text-sm transition-all duration-200",
              "text-gray-900 placeholder-gray-400 dark:bg-gray-800/50 dark:text-gray-50 dark:placeholder-gray-500",
              "hover:shadow-md hover:shadow-gray-500/15 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
              hasError
                ? "border-red-500 focus-visible:ring-red-500 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950"
                : "border-gray-200 focus-visible:ring-primary-500 focus-visible:ring-offset-white dark:border-gray-700 dark:focus-visible:ring-primary-500 dark:focus-visible:ring-offset-gray-950",
              className,
            )}
            {...props}
          />
        </div>
        {hasError && (
          <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
            {meta.error}
          </p>
        )}
      </div>
    );
  },
);

export default PhoneInput;
