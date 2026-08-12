import classNames from "classnames";
import { type FC, useCallback, useEffect, useState } from "react";

interface TOCItem {
  id: string;
  label: string;
  level?: number;
}

interface TOCProps {
  items: TOCItem[];
  className?: string;
}

const TOC: FC<TOCProps> = ({ items, className }) => {
  const [activeId, setActiveId] = useState<string>("");

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);

        if (visible.length > 0) {
          setActiveId(visible[0].target.id);
        }
      },
      { rootMargin: "-80px 0px -70% 0px", threshold: 0 },
    );

    const elements = items
      .map((item) => document.getElementById(item.id))
      .filter(Boolean) as HTMLElement[];

    for (const el of elements) {
      observer.observe(el);
    }
    return () => observer.disconnect();
  }, [items]);

  const scrollTo = useCallback((id: string) => {
    const el = document.getElementById(id);
    if (el) {
      const top = el.getBoundingClientRect().top + window.scrollY - 100;
      window.scrollTo({ top, behavior: "smooth" });
    }
  }, []);

  if (items.length === 0) {
    return null;
  }

  return (
    <nav className={classNames("space-y-0.5", className)}>
      <div className="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
        On this page
      </div>
      {items.map((item) => (
        <button
          key={item.id}
          type="button"
          onClick={() => scrollTo(item.id)}
          className={classNames(
            "block w-full border-l-2 py-1 pl-3 text-left text-sm transition-colors",
            activeId === item.id
              ? "border-primary-500 font-medium text-primary-700 dark:text-primary-300"
              : "border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:text-gray-400 dark:hover:border-gray-600 dark:hover:text-gray-200",
          )}
          style={{ paddingLeft: 12 + (item.level ?? 0) * 12 }}
        >
          {item.label}
        </button>
      ))}
    </nav>
  );
};

export default TOC;
