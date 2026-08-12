import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface ButtonGroupProps {
  children: ReactNode;
  className?: string;
}

const ButtonGroup: FC<ButtonGroupProps> = ({ children, className }) => (
  <div
    className={classNames(
      "isolate inline-flex",
      "[&>*]:relative",
      "[&>*:not(:first-child)]:rounded-l-none",
      "[&>*:not(:last-child)]:rounded-r-none",
      "[&>*:not(:first-child)]:-ml-px",
      "[&>*:active]:z-10 [&>*:focus-visible]:z-10 [&>*:hover]:z-10",
      className,
    )}
  >
    {children}
  </div>
);

export default ButtonGroup;
