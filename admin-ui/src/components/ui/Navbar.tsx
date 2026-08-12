import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface NavbarProps {
  className?: string;
  children: ReactNode;
}

interface NavbarBrandProps {
  className?: string;
  children: ReactNode;
}

interface NavbarContentProps {
  className?: string;
  children: ReactNode;
}

interface NavbarItemProps {
  active?: boolean;
  className?: string;
  children: ReactNode;
}

const Navbar: FC<NavbarProps> & {
  Brand: FC<NavbarBrandProps>;
  Content: FC<NavbarContentProps>;
  Item: FC<NavbarItemProps>;
} = ({ className, children }) => (
  <header
    className={classNames(
      "sticky top-0 z-40 flex h-14 items-center border-b border-gray-200 bg-white/80 px-4 backdrop-blur-md dark:border-gray-700 dark:bg-gray-900/80",
      className,
    )}
  >
    {children}
  </header>
);

const NavbarBrand: FC<NavbarBrandProps> = ({ className, children }) => (
  <div
    className={classNames(
      "mr-4 flex shrink-0 items-center text-sm font-semibold text-gray-900 dark:text-gray-50",
      className,
    )}
  >
    {children}
  </div>
);

const NavbarContent: FC<NavbarContentProps> = ({ className, children }) => (
  <div className={classNames("flex flex-1 items-center gap-1", className)}>
    {children}
  </div>
);

const NavbarItem: FC<NavbarItemProps> = ({ active, className, children }) => (
  <div
    className={classNames(
      "flex items-center rounded-md px-2 py-1 text-sm transition duration-150",
      active
        ? "bg-primary-50 font-medium text-primary-700 dark:bg-primary-900/10 dark:text-primary-300"
        : "text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100",
      className,
    )}
  >
    {children}
  </div>
);

Navbar.Brand = NavbarBrand;
Navbar.Content = NavbarContent;
Navbar.Item = NavbarItem;

export default Navbar;
