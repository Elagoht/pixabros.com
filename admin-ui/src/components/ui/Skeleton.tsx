import classNames from "classnames";
import type { FC } from "react";

interface SkeletonProps {
  className?: string;
  variant?: "text" | "rect" | "circle";
  width?: string | number;
  height?: string | number;
}

const Skeleton: FC<SkeletonProps> = ({
  className,
  variant = "rect",
  width,
  height,
}) => (
  <div
    className={classNames(
      "animate-pulse bg-gray-200 dark:bg-gray-700",
      variant === "circle" && "rounded-full",
      variant === "text" && "rounded",
      variant === "rect" && "rounded-lg",
      className,
    )}
    style={{ width, height }}
  />
);

export default Skeleton;
