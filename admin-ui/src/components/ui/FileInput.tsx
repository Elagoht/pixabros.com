import { IconFile, IconUpload, IconX } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { type FC, useRef, useState } from "react";
import { useI18n } from "@/lib/stores/i18n";

interface FileInputProps {
  name: string;
  label?: string;
  accept?: string;
  className?: string;
}

const FileInput: FC<FileInputProps> = ({ name, label, accept, className }) => {
  const [field, meta, helpers] = useField<File | null>(name);
  const hasError = meta.touched && !!meta.error;
  const inputRef = useRef<HTMLInputElement>(null);
  const file = field.value;
  const { t } = useI18n();
  const [dragOver, setDragOver] = useState(false);

  const inputCls = classNames(
    "w-full rounded-tl-lg border bg-gray-50 px-3 py-2 text-sm transition-all duration-200 ease-out",
    "text-gray-900 dark:bg-gray-800/50 dark:text-gray-50",
    "border-r-0",
    hasError
      ? "border-red-500 hover:ring-2 hover:ring-red-400/40 hover:ring-offset-2 focus-visible:ring-red-500"
      : "border-gray-200 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-2 focus-visible:ring-primary-500",
    "dark:border-gray-700 dark:focus-visible:ring-primary-500 dark:focus-visible:ring-offset-gray-950",
  );

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const dropped = e.dataTransfer.files[0];
    if (dropped) {
      helpers.setValue(dropped);
    }
  };

  return (
    <div className={classNames("w-full", className)}>
      {label && (
        <label
          htmlFor={name}
          className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {label}
        </label>
      )}

      <div className="flex">
        <div className={inputCls}>
          {file ? (
            <div className="flex items-center gap-2">
              <IconFile
                size={16}
                className="shrink-0 text-gray-400 dark:text-gray-500"
              />
              <span className="flex-1 truncate text-gray-700 dark:text-gray-300">
                {file.name}
              </span>
              <span className="shrink-0 text-xs text-gray-400 dark:text-gray-500">
                {file.size < 1024 * 1024
                  ? `${(file.size / 1024).toFixed(1)} KB`
                  : `${(file.size / (1024 * 1024)).toFixed(1)} MB`}
              </span>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  helpers.setValue(null);
                }}
                className="shrink-0 text-gray-400 hover:text-red-500 dark:text-gray-500 dark:hover:text-red-400"
              >
                <IconX size={16} />
              </button>
            </div>
          ) : (
            <span className="text-gray-400 dark:text-gray-500">
              {t("common.noFileChosen")}
            </span>
          )}
        </div>

        <button
          type="button"
          onClick={() => inputRef.current?.click()}
          className={classNames(
            "shrink-0 rounded-tr-lg border bg-gray-50 px-3 py-2 text-sm font-medium transition-all duration-200",
            "border-gray-200 text-gray-600 hover:bg-gray-50 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-2",
            "dark:border-gray-700 dark:bg-gray-800/50 dark:text-gray-300 dark:hover:bg-gray-800 dark:hover:ring-gray-600",
            hasError &&
              "border-red-500 hover:shadow-red-500/20 hover:ring-red-400/40",
          )}
        >
          {t("common.browse")}
        </button>
      </div>

      <div
        onClick={() => inputRef.current?.click()}
        onDragOver={(event) => {
          event.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={handleDrop}
        className={classNames(
          "flex cursor-pointer items-center justify-center gap-2 rounded-b-lg border border-t-0 border-dashed px-3 py-4 text-sm transition-all duration-200",
          dragOver
            ? "border-primary-500 bg-primary-50 text-primary-600 dark:border-primary-500 dark:bg-primary-900/15 dark:text-primary-400"
            : "border-gray-300 text-gray-400 hover:border-primary-400 hover:text-primary-500 hover:bg-gray-50/50 dark:border-gray-600 dark:text-gray-500 dark:hover:border-primary-500 dark:hover:text-primary-400 dark:hover:bg-gray-800/50",
        )}
      >
        <IconUpload size={16} />
        <span>{t("common.dragAndDrop")}</span>
      </div>

      <input
        ref={inputRef}
        id={name}
        type="file"
        accept={accept}
        className="hidden"
        onChange={(e) => {
          const selected = e.target.files?.[0];
          if (selected) {
            helpers.setValue(selected);
          }
          e.target.value = "";
        }}
      />

      {hasError && (
        <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
          {meta.error}
        </p>
      )}
    </div>
  );
};

export default FileInput;
