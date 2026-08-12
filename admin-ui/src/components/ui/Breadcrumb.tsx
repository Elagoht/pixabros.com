import { IconChevronRight } from "@tabler/icons-react";
import { Children, type FC, type ReactNode } from "react";
import { Link, type To } from "react-router-dom";

interface BreadcrumbProps {
  separator?: ReactNode;
  className?: string;
  children: ReactNode;
}

interface BreadcrumbItemProps {
  children: ReactNode;
  to?: To;
}

const BreadcrumbItem: FC<BreadcrumbItemProps> = ({ children, to }) => {
  const content = to ? (
    <Link
      to={to}
      className="text-sm text-gray-400 transition-all duration-200 hover:text-primary-600 dark:text-gray-500 dark:hover:text-primary-400"
    >
      {children}
    </Link>
  ) : (
    <span className="text-sm font-medium text-gray-900 dark:text-gray-50">
      {children}
    </span>
  );
  return <li className="inline-flex items-center">{content}</li>;
};

const Breadcrumb: FC<BreadcrumbProps> & { Item: FC<BreadcrumbItemProps> } = ({
  separator,
  className,
  children,
}) => {
  const items = Children.toArray(children);

  return (
    <nav className={className}>
      <ol className="flex flex-wrap items-center gap-1.5">
        {items.map((child, i) => (
          <span key={i} className="inline-flex items-center gap-1.5">
            {child}
            {i < items.length - 1 && (
              <span className="text-gray-300 dark:text-gray-600">
                {separator ?? <IconChevronRight size={14} />}
              </span>
            )}
          </span>
        ))}
      </ol>
    </nav>
  );
};

Breadcrumb.Item = BreadcrumbItem;

export default Breadcrumb;
