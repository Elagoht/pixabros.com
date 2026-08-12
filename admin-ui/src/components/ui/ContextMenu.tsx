import classNames from "classnames";
import {
  type FC,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

interface ContextMenuItem {
  id: string;
  label?: ReactNode;
  icon?: IconElement;
  disabled?: boolean;
  danger?: boolean;
  separator?: boolean;
  onClick?: () => void;
}

interface ContextMenuProps {
  children: ReactNode;
  items: ContextMenuItem[];
  className?: string;
}

const ContextMenu: FC<ContextMenuProps> = ({ children, items, className }) => {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const menuRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([]);

  const handleContextMenu = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setPos({ x: e.clientX, y: e.clientY });
    setOpen(true);
    setFocusedIndex(0);
  }, []);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setOpen(false);
        setFocusedIndex(-1);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handler = (e: KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setFocusedIndex((prev) => (prev + 1 >= items.length ? 0 : prev + 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setFocusedIndex((prev) => (prev - 1 < 0 ? items.length - 1 : prev - 1));
      } else if (e.key === "Enter" && focusedIndex >= 0) {
        const item = items[focusedIndex];
        if (item && !item.disabled && !item.separator) {
          item.onClick?.();
          setOpen(false);
          setFocusedIndex(-1);
        }
      } else if (e.key === "Escape") {
        setOpen(false);
        setFocusedIndex(-1);
      }
    };
    const onScroll = () => {
      setOpen(false);
      setFocusedIndex(-1);
    };
    document.addEventListener("keydown", handler);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      document.removeEventListener("keydown", handler);
      window.removeEventListener("scroll", onScroll, true);
    };
  }, [open, focusedIndex, items]);

  useEffect(() => {
    if (focusedIndex >= 0 && itemRefs.current[focusedIndex]) {
      itemRefs.current[focusedIndex]?.focus();
    }
  }, [focusedIndex]);

  return (
    <>
      <div
        className={classNames("relative", className)}
        onContextMenu={handleContextMenu}
      >
        {children}
      </div>
      {createPortal(
        <div
          ref={menuRef}
          className={classNames(
            "fixed z-50 min-w-[12rem] rounded-md border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-900",
            "origin-top-left transition-opacity duration-150",
            open ? "opacity-100" : "pointer-events-none opacity-0",
          )}
          style={{
            left: Math.min(pos.x, window.innerWidth - 200),
            top: Math.min(pos.y, window.innerHeight - 200),
          }}
        >
          {items.map((item, i) => {
            if (item.separator) {
              return (
                <div
                  key={item.id}
                  className="my-1 border-t border-gray-200 dark:border-gray-700"
                />
              );
            }

            const isFocused = focusedIndex === i;
            const Icon = item.icon;

            return (
              <button
                key={item.id}
                type="button"
                ref={(el) => {
                  itemRefs.current[i] = el;
                }}
                disabled={item.disabled}
                onClick={() => {
                  if (item.disabled) {
                    return;
                  }
                  item.onClick?.();
                  setOpen(false);
                  setFocusedIndex(-1);
                }}
                className={classNames(
                  "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm outline-none transition duration-100",
                  item.disabled && "pointer-events-none opacity-40",
                  item.danger &&
                    !item.disabled &&
                    isFocused &&
                    "bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400",
                  item.danger &&
                    !item.disabled &&
                    !isFocused &&
                    "text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/10",
                  !(item.danger || item.disabled) &&
                    isFocused &&
                    "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50",
                  !(item.danger || item.disabled || isFocused) &&
                    "text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800",
                )}
              >
                {Icon && <Icon size={16} className="shrink-0" />}
                {item.label}
              </button>
            );
          })}
        </div>,
        document.body,
      )}
    </>
  );
};

export default ContextMenu;
