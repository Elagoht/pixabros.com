import classNames from "classnames";
import type { FC, ReactNode } from "react";

interface PageLayoutProps {
  children: ReactNode;
  className?: string;
  sidebarPosition?: "left" | "right";
}

interface SidebarProps {
  children: ReactNode;
  className?: string;
  width?: "sm" | "md" | "lg";
}

interface ContentProps {
  children: ReactNode;
  className?: string;
}

const sidebarWidths: Record<NonNullable<SidebarProps["width"]>, string> = {
  sm: "w-48 shrink-0",
  md: "w-64 shrink-0",
  lg: "w-80 shrink-0",
};

const PageLayout: FC<PageLayoutProps> & {
  Sidebar: FC<SidebarProps>;
  Content: FC<ContentProps>;
} = ({ children, className, sidebarPosition = "left" }) => (
  <div
    className={classNames(
      "flex min-h-0 w-full gap-6",
      sidebarPosition === "right" && "flex-row-reverse",
      className,
    )}
  >
    {children}
  </div>
);

const Sidebar: FC<SidebarProps> = ({ children, className, width = "md" }) => (
  <aside className={classNames(sidebarWidths[width], className)}>
    {children}
  </aside>
);

const Content: FC<ContentProps> = ({ children, className }) => (
  <main className={classNames("min-w-0 flex-1", className)}>{children}</main>
);

PageLayout.Sidebar = Sidebar;
PageLayout.Content = Content;

export default PageLayout;
