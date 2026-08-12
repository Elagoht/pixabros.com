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

interface CardHeaderProps extends CardSectionProps {
  icon?: IconElement;
}

const Card: FC<CardProps> & {
  Header: FC<CardHeaderProps>;
  Body: FC<CardSectionProps>;
  Footer: FC<CardSectionProps>;
} = ({ children, className }) => (
  <div
    className={classNames(
      "rounded-xl border border-gray-200/60 bg-white shadow-md shadow-gray-500/10 transition-all duration-200",
      "hover:shadow-lg hover:shadow-gray-500/15 hover:border-gray-300/60",
      "dark:border-gray-800 dark:bg-gray-950 dark:shadow-black/40 dark:hover:border-gray-600 dark:hover:shadow-gray-950/60",
      className,
    )}
  >
    {children}
  </div>
);

const Header: FC<CardHeaderProps> = ({ icon: Icon, children, className }) => (
  <div
    className={classNames(
      "flex items-center border-b border-gray-200 px-5 py-4 dark:border-gray-700",
      className,
    )}
  >
    {Icon && (
      <span className="mr-2.5 inline-flex shrink-0 items-center justify-center rounded-md bg-primary-50 p-1.5 text-primary-600 dark:bg-primary-500/10 dark:text-primary-400">
        <Icon size={16} />
      </span>
    )}
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
