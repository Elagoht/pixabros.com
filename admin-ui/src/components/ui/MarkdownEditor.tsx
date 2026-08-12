import classNames from "classnames";
import { useField } from "formik";
import { type FC, useMemo, useState } from "react";
import { useI18n } from "@/lib/stores/i18n";
import { renderMarkdown } from "@/utilities/markdown";
import Button from "./Button";

interface MarkdownEditorProps {
  name: string;
  label?: string;
  rows?: number;
  placeholder?: string;
  className?: string;
}

// A markdown field with a preview tab. Writing and previewing are separate
// tabs rather than a split pane because this sits in a two-column edit page,
// where two side-by-side panes would leave each one too narrow to read.
const MarkdownEditor: FC<MarkdownEditorProps> = ({
  name,
  label,
  rows = 16,
  placeholder,
  className,
}) => {
  const { t } = useI18n();
  const [field, meta] = useField<string>(name);
  const [previewing, setPreviewing] = useState(false);

  const hasError = meta.touched && !!meta.error;
  const source = field.value ?? "";

  // Rendering runs markdown through a sanitiser, so it is kept off the
  // keystroke path and only redone when the preview is actually shown.
  const html = useMemo(
    () => (previewing ? renderMarkdown(source) : ""),
    [previewing, source],
  );

  return (
    <div className={classNames("w-full", className)}>
      <div className="mb-1.5 flex items-end justify-between gap-2">
        {label && (
          <label
            htmlFor={name}
            className="block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {label}
          </label>
        )}
        <div className="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-gray-700 dark:bg-gray-800">
          {[
            { key: "write", label: t("markdown.write"), preview: false },
            { key: "preview", label: t("markdown.preview"), preview: true },
          ].map((tab) => (
            <Button
              key={tab.key}
              variant="ghost"
              size="sm"
              onClick={() => setPreviewing(tab.preview)}
              className={classNames(
                "!rounded-md",
                previewing === tab.preview
                  ? "!bg-white !text-gray-900 !shadow-sm dark:!bg-gray-700 dark:!text-gray-50"
                  : "!text-gray-500 hover:!text-gray-700 dark:!text-gray-400 dark:hover:!text-gray-200",
              )}
            >
              {tab.label}
            </Button>
          ))}
        </div>
      </div>

      {previewing ? (
        <div
          // Height is matched to the textarea so switching tabs does not make
          // the page jump.
          style={{ minHeight: `${rows * 1.5}rem` }}
          className="overflow-auto rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-800/50"
        >
          {source.trim() ? (
            <div
              className="prose-sm max-w-none text-sm text-gray-900 dark:text-gray-50 [&_a]:text-primary-600 [&_a]:underline [&_blockquote]:border-l-2 [&_blockquote]:border-gray-300 [&_blockquote]:pl-3 [&_blockquote]:italic [&_code]:rounded [&_code]:bg-gray-200/70 [&_code]:px-1 [&_code]:font-mono [&_code]:text-xs [&_h1]:mb-2 [&_h1]:mt-3 [&_h1]:text-lg [&_h1]:font-bold [&_h2]:mb-2 [&_h2]:mt-3 [&_h2]:text-base [&_h2]:font-bold [&_h3]:mb-1 [&_h3]:mt-2 [&_h3]:font-semibold [&_hr]:my-3 [&_hr]:border-gray-300 [&_img]:max-w-full [&_img]:rounded [&_li]:my-0.5 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_p]:my-2 [&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-gray-200/70 [&_pre]:p-2 [&_strong]:font-semibold [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5 dark:[&_a]:text-primary-400 dark:[&_blockquote]:border-gray-600 dark:[&_code]:bg-gray-900/70 dark:[&_hr]:border-gray-700 dark:[&_pre]:bg-gray-900/70"
              // The HTML is produced by marked and then sanitised; see
              // utilities/markdown.
              // biome-ignore lint/security/noDangerouslySetInnerHtml: sanitised markdown preview
              dangerouslySetInnerHTML={{ __html: html }}
            />
          ) : (
            <p className="text-sm text-gray-400 dark:text-gray-500">
              {t("markdown.nothingToPreview")}
            </p>
          )}
        </div>
      ) : (
        <textarea
          id={name}
          rows={rows}
          placeholder={placeholder}
          className={classNames(
            "w-full rounded-lg border bg-gray-50 px-3 py-2 font-mono text-sm transition-all duration-200",
            "text-gray-900 placeholder-gray-400 dark:bg-gray-800/50 dark:text-gray-50 dark:placeholder-gray-500",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
            hasError
              ? "border-red-500 focus-visible:ring-red-500"
              : "border-gray-200 focus-visible:ring-primary-500 dark:border-gray-700",
          )}
          {...field}
        />
      )}

      {hasError && (
        <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
          {meta.error}
        </p>
      )}
    </div>
  );
};

export default MarkdownEditor;
