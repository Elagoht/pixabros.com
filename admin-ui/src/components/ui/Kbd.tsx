import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface KbdProps {
  children: ReactNode;
  className?: string;
}

const Kbd: FC<KbdProps> = ({ children, className }) => (
  <kbd
    className={classNames(
      "inline-flex items-center rounded-md border border-b-2 border-gray-300 bg-gradient-to-b from-gray-100 to-gray-200 px-1.5 py-0.5 font-mono text-xs font-medium text-gray-600 shadow-sm shadow-gray-500/10 transition-all duration-200 hover:shadow-md hover:shadow-gray-500/15 dark:border-gray-600 dark:from-gray-800 dark:to-gray-900 dark:text-gray-300",
      className,
    )}
  >
    {children}
  </kbd>
);

export default Kbd;
