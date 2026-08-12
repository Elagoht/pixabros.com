import { IconMail, IconX } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { type FC, useRef, useState } from "react";

interface EmailTagsProps {
  name: string;
  label?: string;
  placeholder?: string;
  className?: string;
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const normalizeEmail = (e: string) => e.trim().toLowerCase();

const EmailTags: FC<EmailTagsProps> = ({
  name,
  label,
  placeholder = "email@example.com",
  className,
}) => {
  const [, meta, helpers] = useField<string[]>(name);
  const hasError = meta.touched && !!meta.error;

  const emails: string[] = Array.isArray(meta.value) ? meta.value : [];

  const [input, setInput] = useState("");
  const [invalid, setInvalid] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const add = (value: string) => {
    const normalized = normalizeEmail(value);
    if (!(normalized && EMAIL_RE.test(normalized))) {
      setInvalid(true);
      return;
    }
    setInvalid(false);
    if (emails.includes(normalized)) {
      setInput("");
      return;
    }
    helpers.setValue([...emails, normalized]);
    setInput("");
  };

  const remove = (index: number) => {
    helpers.setValue(emails.filter((_, i) => i !== index));
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      add(input);
    }
    if (e.key === "Backspace" && !input && emails.length > 0) {
      remove(emails.length - 1);
    }
  };

  const handleBlur = () => {
    if (input.trim()) {
      add(input);
    }
    helpers.setTouched(true);
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
        <IconMail
          size={14}
          className={classNames(
            "shrink-0",
            hasError ? "text-red-400" : "text-gray-400 dark:text-gray-500",
          )}
        />
        {emails.map((email, i) => (
          <span
            key={`${email}-${i}`}
            className="inline-flex items-center gap-0.5 rounded-full bg-gradient-to-b from-blue-100 to-blue-50 px-2 py-1 text-xs font-medium text-blue-700 shadow-sm shadow-blue-500/10 transition-all duration-200 hover:shadow-md hover:shadow-blue-500/15 dark:from-blue-900/30 dark:to-blue-900/20 dark:text-blue-300"
          >
            <button
              type="button"
              tabIndex={-1}
              onClick={() => remove(i)}
              className="text-blue-400 hover:text-red-500 transition-colors duration-150"
            >
              <IconX size={10} />
            </button>
            {email}
          </span>
        ))}
        <input
          ref={inputRef}
          id={name}
          value={input}
          onChange={(e) => {
            setInput(e.target.value);
            setInvalid(false);
          }}
          onKeyDown={handleKeyDown}
          onBlur={handleBlur}
          placeholder={emails.length === 0 ? placeholder : ""}
          className="min-w-[80px] flex-1 bg-transparent outline-none placeholder:text-gray-400 dark:placeholder:text-gray-500"
        />
      </div>
      {invalid && (
        <p className="mt-1.5 text-xs text-amber-500 dark:text-amber-400">
          Geçerli bir e-posta adresi girin
        </p>
      )}
      {hasError && (
        <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
          {meta.error}
        </p>
      )}
    </div>
  );
};

export default EmailTags;
