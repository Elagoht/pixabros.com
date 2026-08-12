import { IconEye, IconEyeOff } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { type ComponentPropsWithoutRef, forwardRef, useState } from "react";

type InputType =
  | "text"
  | "number"
  | "password"
  | "email"
  | "url"
  | "tel"
  | "date";

interface InputProps
  extends Omit<ComponentPropsWithoutRef<"input">, "name" | "type"> {
  name: string;
  type?: InputType;
  label?: string;
  leftIcon?: IconElement;
  rightIcon?: IconElement;
}

const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      name,
      type = "text",
      label,
      leftIcon: LeftIcon,
      rightIcon: RightIcon,
      className,
      ...props
    },
    ref,
  ) => {
    const [field, meta] = useField(name);
    const [showPassword, setShowPassword] = useState(false);

    const hasError = meta.touched && !!meta.error;
    const isPassword = type === "password";
    const inputType = isPassword && showPassword ? "text" : type;

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
          {LeftIcon && (
            <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-gray-400 dark:text-gray-500">
              <LeftIcon size={16} />
            </div>
          )}
          <input
            ref={ref}
            id={name}
            type={inputType}
            className={classNames(
              "w-full rounded-lg border bg-gray-50 px-3 py-2 text-sm transition-all duration-200 ease-out",
              "text-gray-900 placeholder-gray-400 dark:bg-gray-800/50 dark:text-gray-50 dark:placeholder-gray-500",
              hasError
                ? "border-red-500 shadow-md shadow-red-500/20 hover:shadow-lg hover:shadow-red-500/30 hover:ring-2 hover:ring-red-400/40 hover:ring-offset-2 focus-visible:ring-red-500 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950"
                : "border-gray-200 shadow-sm shadow-gray-500/10 hover:shadow-md hover:shadow-gray-500/15 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-2 focus-visible:ring-primary-500 focus-visible:ring-offset-white dark:border-gray-700 dark:focus-visible:ring-primary-500 dark:focus-visible:ring-offset-gray-950",
              LeftIcon && "pl-9",
              (RightIcon || isPassword) && "pr-9",
              className,
            )}
            {...field}
            {...props}
          />
          {isPassword && (
            <button
              type="button"
              tabIndex={-1}
              onClick={() => setShowPassword((v) => !v)}
              className="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 hover:text-primary-600 dark:text-gray-500 dark:hover:text-primary-400 transition-colors duration-150"
            >
              {showPassword ? <IconEyeOff size={16} /> : <IconEye size={16} />}
            </button>
          )}
          {RightIcon && !isPassword && (
            <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 dark:text-gray-500">
              <RightIcon size={16} />
            </div>
          )}
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

export default Input;
