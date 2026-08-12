import { IconChevronDown, IconX } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import {
  type ComponentPropsWithoutRef,
  forwardRef,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";
import Chip from "./Chip";

interface SelectOption {
  label: ReactNode;
  value: string;
}

interface SelectSingleProps
  extends Omit<ComponentPropsWithoutRef<"select">, "name"> {
  name: string;
  label?: string;
  options: SelectOption[];
  placeholder?: string;
  multiple?: false;
}

interface SelectMultiProps
  extends Omit<ComponentPropsWithoutRef<"select">, "name" | "multiple"> {
  name: string;
  label?: string;
  options: SelectOption[];
  placeholder?: string;
  multiple: true;
}

type SelectProps = SelectSingleProps | SelectMultiProps;

const Select = forwardRef<HTMLSelectElement, SelectProps>((props, ref) => {
  const { name, label, options, placeholder, className, ...rest } = props;
  const multiple = "multiple" in props && props.multiple;
  const [field, meta, helpers] = useField({ name, multiple } as {
    name: string;
    multiple?: boolean;
  });
  const hasError = meta.touched && !!meta.error;

  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const selectedMulti =
    multiple && Array.isArray(field.value) ? field.value : [];
  const selectedSingle = multiple ? "" : (field.value ?? "");

  useEffect(() => {
    if (!multiple) {
      return;
    }
    const handler = (e: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [multiple]);

  const toggleMulti = (value: string) => {
    const next = selectedMulti.includes(value)
      ? selectedMulti.filter((v: string) => v !== value)
      : [...selectedMulti, value];
    helpers.setValue(next);
  };

  const removeMulti = (value: string) => {
    helpers.setValue(selectedMulti.filter((v: string) => v !== value));
  };

  const borderCls = classNames(
    "w-full rounded-lg border bg-gray-50 px-3 py-2 text-sm transition-all duration-200 ease-out",
    "text-gray-900 dark:bg-gray-800/50 dark:text-gray-50",
    hasError
      ? "border-red-500 shadow-md shadow-red-500/20 hover:shadow-lg hover:shadow-red-500/30 hover:ring-2 hover:ring-red-400/40 hover:ring-offset-2 focus-visible:ring-red-500 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950"
      : "border-gray-200 shadow-sm shadow-gray-500/10 hover:shadow-md hover:shadow-gray-500/15 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-2 focus-visible:ring-primary-500 focus-visible:ring-offset-white dark:border-gray-700 dark:focus-visible:ring-primary-500 dark:focus-visible:ring-offset-gray-950",
  );

  if (multiple) {
    return (
      <div className="w-full">
        {label && (
          <span className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
            {label}
          </span>
        )}
        <div ref={containerRef} className="relative">
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            className={classNames(
              borderCls,
              "flex min-h-[38px] flex-wrap items-center gap-1 text-left",
              className,
            )}
          >
            {selectedMulti.length > 0 ? (
              selectedMulti.map((v: string) => {
                const opt = options.find((o) => o.value === v);
                return opt ? (
                  <Chip key={v} onRemove={() => removeMulti(v)}>
                    {opt.label}
                  </Chip>
                ) : null;
              })
            ) : (
              <span className="text-gray-400 dark:text-gray-500">
                {placeholder ?? "Select..."}
              </span>
            )}
            <span className="ml-auto flex shrink-0 items-center gap-1">
              {selectedMulti.length > 0 && (
                <span
                  onClick={(e) => {
                    e.stopPropagation();
                    helpers.setValue([]);
                  }}
                  className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
                >
                  <IconX size={14} />
                </span>
              )}
              <IconChevronDown
                size={16}
                className="text-gray-400 dark:text-gray-500"
              />
            </span>
          </button>

          {open && (
            <div className="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-xl border border-gray-200/60 bg-white py-1 shadow-lg shadow-gray-500/15 dark:border-gray-700/60 dark:bg-gray-900">
              {options.map((opt) => {
                const checked = selectedMulti.includes(opt.value);
                return (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => toggleMulti(opt.value)}
                    className={classNames(
                      "flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-all duration-200",
                      checked
                        ? "bg-primary-50 text-primary-700 shadow-sm shadow-primary-500/10 dark:bg-primary-900/15 dark:text-primary-300"
                        : "text-gray-700 hover:bg-gray-50 hover:shadow-sm hover:shadow-gray-500/10 dark:text-gray-300 dark:hover:bg-gray-800",
                    )}
                  >
                    <span
                      className={classNames(
                        "flex h-4 w-4 shrink-0 items-center justify-center rounded border transition",
                        checked
                          ? "border-primary-500 bg-primary-500"
                          : "border-gray-300 dark:border-gray-600",
                      )}
                    >
                      {checked && (
                        <span className="text-[10px] text-white">✓</span>
                      )}
                    </span>
                    {opt.label}
                  </button>
                );
              })}
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
  }

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
        <select
          ref={ref}
          id={name}
          className={classNames(
            borderCls,
            "appearance-none pr-9",
            !selectedSingle && "text-gray-400 dark:text-gray-500",
            className,
          )}
          {...field}
          {...rest}
        >
          {placeholder && (
            <option value="" disabled>
              {placeholder}
            </option>
          )}
          {options.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
        <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400 dark:text-gray-500">
          <IconChevronDown size={16} />
        </div>
      </div>
      {hasError && (
        <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
          {meta.error}
        </p>
      )}
    </div>
  );
});

export default Select;
