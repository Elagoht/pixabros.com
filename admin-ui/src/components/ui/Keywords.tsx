import { IconX } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { type FC, useRef, useState } from "react";

interface KeywordsProps {
  name: string;
  label?: string;
  placeholder?: string;
  separator?: string;
  output?: "string" | "array";
  className?: string;
}

const Keywords: FC<KeywordsProps> = ({
  name,
  label,
  placeholder = "Add keyword...",
  separator = ", ",
  output = "string",
  className,
}) => {
  const [field, meta, helpers] = useField<string | string[]>(name);
  const hasError = meta.touched && !!meta.error;

  const keywords: string[] =
    output === "string"
      ? typeof field.value === "string" && field.value
        ? field.value.split(separator).filter(Boolean)
        : []
      : Array.isArray(field.value)
        ? field.value
        : [];

  const [input, setInput] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const commitValue = (next: string[]) => {
    helpers.setValue(output === "string" ? next.join(separator) : next);
  };

  // A value may carry several keywords at once -- pasted from a document or
  // a spreadsheet column -- so it is always split before being added. Tabs
  // and newlines count as separators too, since that is what a pasted list or
  // spreadsheet cell range actually contains.
  const splitValues = (raw: string): string[] => {
    let parts = [raw];
    for (const token of [separator.trim(), "\n", "\r", "\t"]) {
      if (token) {
        parts = parts.flatMap((part) => part.split(token));
      }
    }
    return parts.map((part) => part.trim()).filter(Boolean);
  };

  const add = (value: string) => {
    const incoming = splitValues(value);
    const next = [...keywords];
    for (const keyword of incoming) {
      if (!next.includes(keyword)) {
        next.push(keyword);
      }
    }
    // Clear the field even when everything pasted was a duplicate, so the
    // text does not linger looking like it failed to register.
    setInput("");
    if (next.length !== keywords.length) {
      commitValue(next);
    }
  };

  const remove = (index: number) => {
    commitValue(keywords.filter((_, i) => i !== index));
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === separator.trim()) {
      e.preventDefault();
      add(input);
    }
    if (e.key === "Backspace" && !input && keywords.length > 0) {
      remove(keywords.length - 1);
    }
  };

  const handleBlur = () => {
    if (input.trim()) {
      add(input);
    }
  };

  // Pasting a separated list turns straight into chips. A paste with no
  // separator in it is left to the browser so it can be edited before being
  // committed, exactly like typing.
  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    const pasted = e.clipboardData.getData("text");
    if (splitValues(pasted).length < 2) {
      return;
    }
    e.preventDefault();
    add(input + pasted);
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
      <div
        onClick={() => inputRef.current?.focus()}
        className={classNames(
          "flex flex-wrap items-center gap-1 rounded-lg border bg-white px-3 py-2 text-sm transition-all duration-200",
          "text-gray-900 dark:bg-gray-900 dark:text-gray-50",
          "hover:shadow-md hover:shadow-gray-500/15 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1",
          "focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2",
          hasError
            ? "border-red-500 focus-within:ring-red-500 focus-within:ring-offset-white dark:focus-within:ring-offset-gray-950"
            : "border-gray-200 focus-within:ring-primary-500 focus-within:ring-offset-white dark:border-gray-700 dark:focus-within:ring-offset-gray-950",
        )}
      >
        {keywords.map((kw, i) => (
          <span
            key={`${kw}-${i}`}
            className="inline-flex items-center gap-0.5 rounded-full bg-gradient-to-b from-primary-100 to-primary-50 px-2 py-1 text-xs font-medium text-primary-700 shadow-sm shadow-primary-500/10 transition-all duration-200 hover:shadow-md hover:shadow-primary-500/15 dark:from-primary-900/30 dark:to-primary-900/20 dark:text-primary-300"
          >
            <button
              type="button"
              tabIndex={-1}
              onClick={() => remove(i)}
              className="text-primary-400 hover:text-red-500 transition-colors duration-150"
            >
              <IconX size={10} />
            </button>
            {kw}
          </span>
        ))}
        <input
          ref={inputRef}
          id={name}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          onPaste={handlePaste}
          onBlur={handleBlur}
          placeholder={keywords.length === 0 ? placeholder : ""}
          className="min-w-[80px] flex-1 bg-transparent outline-none placeholder:text-gray-400 dark:placeholder:text-gray-500"
        />
      </div>
      {hasError && (
        <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
          {meta.error}
        </p>
      )}
    </div>
  );
};

export default Keywords;
