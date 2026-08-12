import classNames from "classnames";
import type { FC } from "react";

interface SeparatorProps {
  orientation?: "horizontal" | "vertical";
  className?: string;
}

const Separator: FC<SeparatorProps> = ({
  orientation = "horizontal",
  className,
}) => (
  <div
    role="separator"
    className={classNames(
      "shrink-0 bg-gray-200 dark:bg-gray-700",
      orientation === "horizontal" ? "h-px w-full" : "h-full w-px",
      className,
    )}
  />
);

export default Separator;
