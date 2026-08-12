import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface ContainerProps {
  children: ReactNode;
  className?: string;
  size?: "sm" | "md" | "lg" | "xl" | "full";
}

const maxWidths: Record<NonNullable<ContainerProps["size"]>, string> = {
  sm: "max-w-2xl",
  md: "max-w-4xl",
  lg: "max-w-6xl",
  xl: "max-w-7xl",
  full: "max-w-full",
};

const Container: FC<ContainerProps> = ({
  children,
  className,
  size = "lg",
}) => (
  <div
    className={classNames(
      "mx-auto w-full px-4 sm:px-6",
      maxWidths[size],
      className,
    )}
  >
    {children}
  </div>
);

export default Container;
