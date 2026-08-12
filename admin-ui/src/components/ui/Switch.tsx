import classNames from "classnames";
import { useField } from "formik";
import type { FC } from "react";

interface SwitchProps {
  name: string;
  label?: string;
  className?: string;
}

const Switch: FC<SwitchProps> = ({ name, label, className }) => {
  const [field, , helpers] = useField<boolean>(name);

  return (
    <label
      className={classNames(
        "flex cursor-pointer items-center gap-2",
        className,
      )}
    >
      <button
        type="button"
        role="switch"
        aria-checked={field.value}
        onClick={() => helpers.setValue(!field.value)}
        className={classNames(
          "relative inline-flex h-6 w-10 shrink-0 rounded-full transition-all duration-200 ease-out",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950",
          field.value
            ? "bg-primary-500 hover:ring-2 hover:ring-primary-400/40"
            : "bg-gray-300 dark:bg-gray-600",
        )}
      >
        <span
          className={classNames(
            "pointer-events-none inline-block h-4 w-4 translate-y-1 rounded-full bg-white transition-transform duration-200 ease-out",
            field.value ? "translate-x-5" : "translate-x-1",
          )}
        />
      </button>
      {label && (
        <span className="select-none text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </span>
      )}
    </label>
  );
};

export default Switch;
