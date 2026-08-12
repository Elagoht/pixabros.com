import { IconChevronDown } from "@tabler/icons-react";
import classNames from "classnames";
import {
  type FC,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";

interface MenuBarItem {
  id: string;
  label: ReactNode;
  children?: {
    id: string;
    label: ReactNode;
    icon?: IconElement;
    disabled?: boolean;
    shortcut?: string;
    onClick?: () => void;
  }[];
}

interface MenuBarProps {
  items: MenuBarItem[];
  className?: string;
}

const MenuBar: FC<MenuBarProps> = ({ items, className }) => {
  const [activeId, setActiveId] = useState<string | null>(null);
  const [focusIdx, setFocusIdx] = useState(0);
  const menuRef = useRef<HTMLDivElement>(null);

  const activeItem = items.find((i) => i.id === activeId);
  const hasChildren = activeItem?.children && activeItem.children.length > 0;

  const select = useCallback((id: string | null) => {
    setActiveId(id);
    setFocusIdx(0);
  }, []);

  useEffect(() => {
    if (!activeId) {
      return;
    }
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        select(null);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [activeId, select]);

  useEffect(() => {
    if (!(activeId && activeItem?.children)) {
      return;
    }
    const handler = (e: KeyboardEvent) => {
      const children = activeItem.children;
      if (!children) {
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setFocusIdx((i) => Math.min(i + 1, children.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setFocusIdx((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        const item = children[focusIdx];
        if (item && !item.disabled) {
          item.onClick?.();
          select(null);
        }
      } else if (e.key === "Escape") {
        select(null);
      } else if (e.key === "ArrowRight") {
        e.preventDefault();
        const idx = items.findIndex((i) => i.id === activeId);
        if (idx < items.length - 1) {
          select(items[idx + 1].id);
        }
      } else if (e.key === "ArrowLeft") {
        e.preventDefault();
        const idx = items.findIndex((i) => i.id === activeId);
        if (idx > 0) {
          select(items[idx - 1].id);
        }
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [activeId, activeItem, focusIdx, items, select]);

  return (
    <div ref={menuRef} className={classNames("relative", className)}>
      <div className="flex items-center rounded-lg border border-gray-200 bg-white px-1 dark:border-gray-700 dark:bg-gray-950">
        {items.map((item) => (
          <button
            key={item.id}
            type="button"
            onClick={() => select(activeId === item.id ? null : item.id)}
            onMouseEnter={() => activeId && select(item.id)}
            className={classNames(
              "flex items-center gap-1 rounded px-3 py-1.5 text-sm transition-colors",
              activeId === item.id
                ? "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50"
                : "text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-900 dark:hover:text-gray-200",
            )}
          >
            {item.label}
            {item.children && item.children.length > 0 && (
              <IconChevronDown size={12} className="text-gray-400" />
            )}
          </button>
        ))}
      </div>

      {activeId && hasChildren && (
        <div className="absolute left-0 top-full z-50 mt-1 min-w-[14rem] overflow-hidden rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-900">
          {activeItem?.children?.map((child, idx) => (
            <button
              key={child.id}
              type="button"
              disabled={child.disabled}
              onClick={() => {
                if (!child.disabled) {
                  child.onClick?.();
                  select(null);
                }
              }}
              onMouseEnter={() => setFocusIdx(idx)}
              className={classNames(
                "flex w-full items-center gap-3 px-3 py-2 text-left text-sm transition-colors",
                child.disabled
                  ? "cursor-not-allowed text-gray-300 dark:text-gray-600"
                  : focusIdx === idx
                    ? "bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300"
                    : "text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800",
              )}
            >
              {child.icon && (
                <child.icon
                  size={16}
                  className="shrink-0 text-gray-400 dark:text-gray-500"
                />
              )}
              <span className="flex-1">{child.label}</span>
              {child.shortcut && (
                <span className="text-xs text-gray-400 dark:text-gray-500">
                  {child.shortcut}
                </span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default MenuBar;
