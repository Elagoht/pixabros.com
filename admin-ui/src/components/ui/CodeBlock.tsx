import { IconCheck, IconCopy, IconFileCode } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useState } from "react";

interface CodeBlockProps {
  code: string;
  language?: string;
  filename?: string;
  maxHeight?: string | number;
  className?: string;
}

const CodeBlock: FC<CodeBlockProps> = ({
  code,
  language,
  filename,
  maxHeight = 400,
  className,
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      const ta = document.createElement("textarea");
      ta.value = code;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      document.body.removeChild(ta);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div
      className={classNames(
        "overflow-hidden rounded-lg border border-gray-200 bg-gray-950 dark:border-gray-700",
        className,
      )}
    >
      <div className="flex items-center justify-between border-b border-gray-800 px-4 py-2">
        <div className="flex items-center gap-2 text-xs text-gray-400">
          <IconFileCode size={14} />
          {filename ? (
            <span className="font-medium text-gray-300">{filename}</span>
          ) : language ? (
            <span>{language}</span>
          ) : null}
        </div>
        <button
          type="button"
          onClick={handleCopy}
          className="flex items-center gap-1 rounded px-2 py-1 text-xs text-gray-400 transition-colors hover:bg-gray-800 hover:text-gray-200"
        >
          {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="overflow-x-auto p-4" style={{ maxHeight }}>
        <code className="text-sm leading-relaxed text-gray-200">{code}</code>
      </pre>
    </div>
  );
};

export default CodeBlock;
