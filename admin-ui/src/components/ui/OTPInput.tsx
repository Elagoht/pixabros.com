import classNames from "classnames";
import { type ClipboardEvent, type FC, useCallback, useRef } from "react";

interface OTPInputProps {
  digits?: number;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  className?: string;
}

const OTPInput: FC<OTPInputProps> = ({
  digits = 6,
  value,
  onChange,
  disabled,
  className,
}) => {
  const inputsRef = useRef<(HTMLInputElement | null)[]>([]);

  const getChars = useCallback(
    (v: string) => {
      const chars = v.replace(/\s/g, "").split("").slice(0, digits);
      while (chars.length < digits) {
        chars.push("");
      }
      return chars;
    },
    [digits],
  );

  const handleChange = useCallback(
    (index: number, char: string) => {
      const chars = getChars(value);
      chars[index] = char.slice(-1);
      onChange(chars.join(""));
      if (char && index < digits - 1) {
        inputsRef.current[index + 1]?.focus();
      }
    },
    [value, digits, onChange, getChars],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent, index: number) => {
      if (e.key === "Backspace") {
        const chars = getChars(value);
        if (!chars[index] && index > 0) {
          inputsRef.current[index - 1]?.focus();
        }
      } else if (e.key === "ArrowLeft" && index > 0) {
        inputsRef.current[index - 1]?.focus();
      } else if (e.key === "ArrowRight" && index < digits - 1) {
        inputsRef.current[index + 1]?.focus();
      }
    },
    [value, digits, getChars],
  );

  const handlePaste = useCallback(
    (e: ClipboardEvent) => {
      const pasted = e.clipboardData
        .getData("text")
        .replace(/\s/g, "")
        .replace(/\D/g, "")
        .slice(0, digits);

      if (pasted) {
        e.preventDefault();
        onChange(pasted.padEnd(digits, "").slice(0, digits));

        const nextIndex = Math.min(pasted.length, digits - 1);
        inputsRef.current[nextIndex]?.focus();
      }
    },
    [digits, onChange],
  );

  const chars = getChars(value);

  return (
    <div className={classNames("flex items-center gap-2", className)}>
      {Array.from({ length: digits }).map((_, i) => (
        <input
          key={i}
          ref={(el) => {
            inputsRef.current[i] = el;
          }}
          type="text"
          inputMode="numeric"
          maxLength={1}
          autoComplete="one-time-code"
          value={chars[i]}
          onChange={(e) => handleChange(i, e.target.value)}
          onKeyDown={(e) => handleKeyDown(e, i)}
          onPaste={i === 0 ? handlePaste : undefined}
          disabled={disabled}
          className={classNames(
            "h-12 w-10 rounded-md border text-center text-lg font-semibold transition duration-150 ease-out",
            "bg-gray-50 text-gray-900 shadow-inner dark:bg-gray-800/50 dark:text-gray-50",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2",
            "ring-offset-white dark:ring-offset-gray-950",
            disabled
              ? "border-gray-200 bg-gray-50 text-gray-400 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-500"
              : "border-gray-200 dark:border-gray-700",
          )}
        />
      ))}
    </div>
  );
};

export default OTPInput;
