import classNames from "classnames";
import { useField } from "formik";
import { type ComponentPropsWithoutRef, forwardRef } from "react";

interface TextareaProps
  extends Omit<ComponentPropsWithoutRef<"textarea">, "name"> {
  name: string;
  label?: string;
}

const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ name, label, className, rows = 4, ...props }, ref) => {
    const [field, meta] = useField(name);
    const hasError = meta.touched && !!meta.error;

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
        <textarea
          ref={ref}
          id={name}
          rows={rows}
          className={classNames(
            "w-full resize-none rounded-lg border bg-gray-50 px-3 py-2 text-sm transition-all duration-200 ease-out",
            "text-gray-900 placeholder-gray-400 dark:bg-gray-800/50 dark:text-gray-50 dark:placeholder-gray-500",
            hasError
              ? "border-red-500 shadow-md shadow-red-500/20 hover:shadow-lg hover:shadow-red-500/30 hover:ring-2 hover:ring-red-400/40 hover:ring-offset-2 focus-visible:ring-red-500 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950"
              : "border-gray-200 shadow-sm shadow-gray-500/10 hover:shadow-md hover:shadow-gray-500/15 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-2 focus-visible:ring-primary-500 focus-visible:ring-offset-white dark:border-gray-700 dark:focus-visible:ring-primary-500 dark:focus-visible:ring-offset-gray-950",
            className,
          )}
          {...field}
          {...props}
        />
        {hasError && (
          <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
            {meta.error}
          </p>
        )}
      </div>
    );
  },
);

export default Textarea;
