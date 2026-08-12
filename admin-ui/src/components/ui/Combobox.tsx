import { IconChevronDown, IconSearch, IconX } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { type FC, type ReactNode, useEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/stores/i18n";
import Chip from "./Chip";

interface ComboboxOption {
  label: ReactNode;
  value: string;
}

interface ComboboxProps {
  name: string;
  label?: string;
  options: ComboboxOption[];
  placeholder?: string;
  multiple?: boolean;
  disabled?: boolean;
  className?: string;
}

const Combobox: FC<ComboboxProps> = ({
  name,
  label,
  options,
  placeholder,
  multiple = false,
  disabled = false,
  className,
}) => {
  const [, meta, helpers] = useField(name);
  const hasError = meta.touched && !!meta.error;
  const { t } = useI18n();

  const selectedSingle = multiple ? "" : (meta.value ?? "");
  const selectedMulti = multiple
    ? Array.isArray(meta.value)
      ? meta.value
      : []
    : [];

  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const filtered = query
    ? options.filter((opt) =>
        String(opt.label).toLowerCase().includes(query.toLowerCase()),
      )
    : options;

  const selectedLabel = options.find((o) => o.value === selectedSingle)?.label;

  useEffect(() => {
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
  }, []);

  const toggle = (value: string) => {
    if (!multiple) {
      helpers.setValue(value);
      setQuery("");
      setOpen(false);
      return;
    }
    const next = selectedMulti.includes(value)
      ? selectedMulti.filter((v: string) => v !== value)
      : [...selectedMulti, value];
    helpers.setValue(next);
  };

  const removeMulti = (value: string) => {
    helpers.setValue(selectedMulti.filter((v: string) => v !== value));
  };

  const isSelected = (value: string) =>
    multiple ? selectedMulti.includes(value) : value === selectedSingle;

  const inputBorderCls = classNames(
    "flex items-center rounded-lg border bg-gray-50 px-3 py-2 text-sm transition-all duration-200",
    "text-gray-900 dark:bg-gray-800/50 dark:text-gray-50",
    disabled
      ? "cursor-not-allowed opacity-50"
      : "cursor-text hover:shadow-md hover:shadow-gray-500/15 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1",
    "focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2",
    hasError
      ? "border-red-500 focus-within:ring-red-500 focus-within:ring-offset-white dark:focus-within:ring-offset-gray-950"
      : "border-gray-200 focus-within:ring-primary-500 focus-within:ring-offset-white dark:border-gray-700 focus-within:ring-primary-500 dark:focus-within:ring-offset-gray-950",
  );

  return (
    <div className={classNames("w-full", className)}>
      {label && (
        <span className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </span>
      )}
      <div ref={containerRef} className="relative">
        <div
          onClick={() => {
            if (disabled) {
              return;
            }
            setOpen(true);
            inputRef.current?.focus();
          }}
          className={classNames(
            inputBorderCls,
            multiple && "min-h-[38px] flex-wrap gap-1",
          )}
        >
          <IconSearch
            size={16}
            className="mr-1 shrink-0 text-gray-400 dark:text-gray-500"
          />

          {multiple &&
            selectedMulti.map((v: string) => {
              const opt = options.find((o) => o.value === v);
              return opt ? (
                <Chip key={v} onRemove={() => removeMulti(v)}>
                  {opt.label}
                </Chip>
              ) : null;
            })}

          {open ? (
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={
                multiple
                  ? "Search..."
                  : typeof selectedLabel === "string"
                    ? selectedLabel
                    : (placeholder ?? "Search...")
              }
              className="min-w-[60px] flex-1 bg-transparent outline-none placeholder:text-gray-400 dark:placeholder:text-gray-500"
            />
          ) : multiple ? (
            selectedMulti.length === 0 && (
              <span className="flex-1 text-gray-400 dark:text-gray-500">
                {placeholder ?? "Select..."}
              </span>
            )
          ) : (
            <span
              className={classNames(
                "flex-1 truncate",
                !selectedSingle && "text-gray-400 dark:text-gray-500",
              )}
            >
              {selectedLabel ?? placeholder ?? "Select..."}
            </span>
          )}

          {!multiple && selectedSingle && !open && (
            <button
              type="button"
              tabIndex={-1}
              disabled={disabled}
              onClick={(e) => {
                e.stopPropagation();
                helpers.setValue("");
              }}
              className="ml-1 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 disabled:opacity-0"
            >
              <IconX size={14} />
            </button>
          )}
          {multiple && selectedMulti.length > 0 && !open && (
            <button
              type="button"
              tabIndex={-1}
              disabled={disabled}
              onClick={(e) => {
                e.stopPropagation();
                helpers.setValue([]);
              }}
              className="ml-1 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 disabled:opacity-0"
            >
              <IconX size={14} />
            </button>
          )}
          <IconChevronDown
            size={16}
            className="ml-1 shrink-0 text-gray-400 dark:text-gray-500"
          />
        </div>

        {open && filtered.length > 0 && (
          <div className="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-xl border border-gray-200/60 bg-white py-1 shadow-lg shadow-gray-500/15 dark:border-gray-700/60 dark:bg-gray-900">
            {filtered.map((opt) => (
              <button
                key={opt.value}
                type="button"
                onClick={() => toggle(opt.value)}
                className={classNames(
                  "flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-all duration-200",
                  isSelected(opt.value)
                    ? "bg-primary-50 text-primary-700 shadow-sm shadow-primary-500/10 dark:bg-primary-900/15 dark:text-primary-300"
                    : "text-gray-700 hover:bg-gray-50 hover:shadow-sm hover:shadow-gray-500/10 dark:text-gray-300 dark:hover:bg-gray-800",
                )}
              >
                {multiple && (
                  <span
                    className={classNames(
                      "flex h-4 w-4 shrink-0 items-center justify-center rounded border transition",
                      isSelected(opt.value)
                        ? "border-primary-500 bg-primary-500"
                        : "border-gray-300 dark:border-gray-600",
                    )}
                  >
                    {isSelected(opt.value) && (
                      <span className="text-[10px] text-white">✓</span>
                    )}
                  </span>
                )}
                {opt.label}
              </button>
            ))}
          </div>
        )}

        {open && filtered.length === 0 && (
          <div className="absolute z-50 mt-1 w-full rounded-md border border-gray-200 bg-white px-3 py-4 text-center text-sm text-gray-400 shadow-lg dark:border-gray-700 dark:bg-gray-900 dark:text-gray-500">
            {t("common.noResults")}
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
};

export default Combobox;
