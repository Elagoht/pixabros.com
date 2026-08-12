import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import classNames from "classnames";
import type { FC } from "react";

interface PaginationProps {
  page: number;
  totalPages: number;
  onChange: (page: number) => void;
  siblingCount?: number;
  className?: string;
}

const range = (start: number, end: number) =>
  Array.from({ length: end - start + 1 }, (_, i) => start + i);

const buildPages = (
  current: number,
  total: number,
  siblings: number,
): (number | "...")[] => {
  if (total <= 7) {
    return range(1, total);
  }

  const left = Math.max(2, current - siblings);
  const right = Math.min(total - 1, current + siblings);

  const showLeftDots = left > 2;
  const showRightDots = right < total - 1;

  if (!showLeftDots && showRightDots) {
    return [...range(1, 4), "...", total];
  }
  if (showLeftDots && !showRightDots) {
    return [1, "...", ...range(total - 3, total)];
  }

  return [1, "...", ...range(left, right), "...", total];
};

const Pagination: FC<PaginationProps> = ({
  page,
  totalPages,
  onChange,
  siblingCount = 1,
  className,
}) => {
  if (totalPages <= 1) {
    return null;
  }

  const pages = buildPages(page, totalPages, siblingCount);

  const btnCls = (active: boolean) =>
    classNames(
      "flex h-8 w-8 items-center justify-center rounded-lg text-xs font-medium transition-all duration-200",
      active
        ? "bg-primary-500 text-white"
        : "text-gray-600 hover:bg-gray-100 hover:ring-1 hover:ring-gray-200/50 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:ring-gray-700/50",
    );

  return (
    <nav className={classNames("flex items-center gap-1", className)}>
      <button
        type="button"
        disabled={page <= 1}
        onClick={() => onChange(page - 1)}
        className={classNames(
          "flex h-8 w-8 items-center justify-center rounded-md transition duration-100",
          page <= 1
            ? "pointer-events-none text-gray-300 dark:text-gray-600"
            : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800",
        )}
      >
        <IconChevronLeft size={16} />
      </button>

      {pages.map((p, i) =>
        p === "..." ? (
          <span
            key={`dots-${i}`}
            className="flex h-8 w-8 items-center justify-center text-xs text-gray-400"
          >
            ...
          </span>
        ) : (
          <button
            key={p}
            type="button"
            onClick={() => onChange(p)}
            className={btnCls(p === page)}
          >
            {p}
          </button>
        ),
      )}

      <button
        type="button"
        disabled={page >= totalPages}
        onClick={() => onChange(page + 1)}
        className={classNames(
          "flex h-8 w-8 items-center justify-center rounded-md transition duration-100",
          page >= totalPages
            ? "pointer-events-none text-gray-300 dark:text-gray-600"
            : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800",
        )}
      >
        <IconChevronRight size={16} />
      </button>
    </nav>
  );
};

export default Pagination;
