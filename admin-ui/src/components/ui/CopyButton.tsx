import { IconCheck, IconCopy } from "@tabler/icons-react";
import classNames from "classnames";
import {
  type ComponentPropsWithoutRef,
  type FC,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

interface CopyButtonProps
  extends Omit<ComponentPropsWithoutRef<"button">, "onClick" | "type"> {
  value: string;
  timeout?: number;
}

const CopyButton: FC<CopyButtonProps> = ({
  value,
  timeout = 2000,
  className,
  children,
  ...props
}) => {
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => {
      clearTimeout(timerRef.current);
    };
  }, []);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      timerRef.current = setTimeout(() => setCopied(false), timeout);
    } catch {
      const ta = document.createElement("textarea");
      ta.value = value;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      document.body.removeChild(ta);
      setCopied(true);
      timerRef.current = setTimeout(() => setCopied(false), timeout);
    }
  }, [value, timeout]);

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={classNames(
        "inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium transition duration-150 ease-out",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2",
        "ring-offset-white dark:ring-offset-gray-950",
        copied
          ? "bg-green-100 text-green-700 dark:bg-green-800/40 dark:text-green-300"
          : "bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700",
        className,
      )}
      {...props}
    >
      {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
      {children ?? (copied ? "Copied" : "Copy")}
    </button>
  );
};

export default CopyButton;
