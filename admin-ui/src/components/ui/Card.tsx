import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface CardProps {
  children: ReactNode;
  className?: string;
}

interface CardSectionProps {
  children: ReactNode;
  className?: string;
}

const Card: FC<CardProps> & {
  Header: FC<CardSectionProps>;
  Body: FC<CardSectionProps>;
  Footer: FC<CardSectionProps>;
} = ({ children, className }) => (
  <div
    className={classNames(
      "rounded-xl border border-gray-200/60 bg-white shadow-md shadow-gray-500/10 transition-all duration-200",
      "hover:shadow-lg hover:shadow-gray-500/15 hover:border-gray-300/60",
      "dark:border-gray-700 dark:bg-gray-900 dark:shadow-gray-950/50 dark:hover:border-gray-600 dark:hover:shadow-gray-950/60",
      className,
    )}
  >
    {children}
  </div>
);

const Header: FC<CardSectionProps> = ({ children, className }) => (
  <div
    className={classNames(
      "flex items-center border-b border-gray-200 px-5 py-4 dark:border-gray-700",
      className,
    )}
  >
    {children}
  </div>
);

const Body: FC<CardSectionProps> = ({ children, className }) => (
  <div className={classNames("px-5 py-4", className)}>{children}</div>
);

const Footer: FC<CardSectionProps> = ({ children, className }) => (
  <div
    className={classNames(
      "flex items-center border-t border-gray-200 px-5 py-4 dark:border-gray-700",
      className,
    )}
  >
    {children}
  </div>
);

Card.Header = Header;
Card.Body = Body;
Card.Footer = Footer;

export default Card;
