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

interface DropdownItem {
  id: string;
  label?: ReactNode;
  icon?: IconElement;
  disabled?: boolean;
  danger?: boolean;
  separator?: boolean;
  onClick?: () => void;
}

interface DropdownProps {
  trigger: ReactNode;
  items: DropdownItem[];
  align?: "left" | "right";
  className?: string;
}

const Dropdown: FC<DropdownProps> = ({
  trigger,
  items,
  align = "left",
  className,
}) => {
  const [open, setOpen] = useState(false);
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const [pos, setPos] = useState<{
    x: number;
    y: number;
    width: number;
    height: number;
  } | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<(HTMLButtonElement | null)[]>([]);

  const updatePos = useCallback(() => {
    const rect = containerRef.current?.getBoundingClientRect();
    if (rect) {
      setPos({
        x: rect.left,
        y: rect.top,
        width: rect.width,
        height: rect.height,
      });
    }
  }, []);

  useEffect(() => {
    if (!open) {
      setPos(null);
      return;
    }
    updatePos();
    const handler = (e: MouseEvent) => {
      const target = e.target as Node;
      if (
        containerRef.current &&
        !containerRef.current.contains(target) &&
        !menuRef.current?.contains(target)
      ) {
        setOpen(false);
        setFocusedIndex(-1);
      }
    };
    const onScroll = () => {
      setOpen(false);
      setFocusedIndex(-1);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        setFocusedIndex(-1);
      }
    };
    document.addEventListener("mousedown", handler);
    window.addEventListener("scroll", onScroll, true);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", handler);
      window.removeEventListener("scroll", onScroll, true);
      document.removeEventListener("keydown", onKey);
    };
  }, [open, updatePos]);

  useEffect(() => {
    if (focusedIndex >= 0 && itemRefs.current[focusedIndex]) {
      itemRefs.current[focusedIndex]?.focus();
    }
  }, [focusedIndex]);

  const firstFocusableIndex = items.findIndex((item) => !item.separator);

  const toggle = useCallback(() => {
    setOpen((prev) => {
      const next = !prev;
      setFocusedIndex(next ? firstFocusableIndex : -1);
      return next;
    });
  }, [firstFocusableIndex]);

  const navigateItem = useCallback(
    (direction: "next" | "prev") => {
      setFocusedIndex((prev) => {
        const step = direction === "next" ? 1 : -1;
        let next = prev;
        for (const _ of items) {
          next = next + step;
          if (next >= items.length) {
            next = 0;
          }
          if (next < 0) {
            next = items.length - 1;
          }
          if (!items[next].separator) {
            return next;
          }
        }
        return prev;
      });
    },
    [items],
  );

  const selectItem = useCallback(() => {
    const item = items[focusedIndex];
    if (item && !item.disabled && !item.separator) {
      item.onClick?.();
      setOpen(false);
      setFocusedIndex(-1);
    }
  }, [focusedIndex, items]);

  const handleTriggerKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown" || e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        if (!open) {
          setOpen(true);
          setFocusedIndex(firstFocusableIndex >= 0 ? firstFocusableIndex : 0);
        }
      } else if (e.key === "Escape") {
        setOpen(false);
        setFocusedIndex(-1);
      }
    },
    [open, firstFocusableIndex],
  );

  const handleMenuKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        navigateItem("next");
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        navigateItem("prev");
      } else if (e.key === "Enter") {
        e.preventDefault();
        selectItem();
      } else if (e.key === "Escape") {
        e.preventDefault();
        setOpen(false);
        setFocusedIndex(-1);
      }
    },
    [navigateItem, selectItem],
  );

  const menuWidth = 192;

  return (
    <>
      <div
        ref={containerRef}
        className={classNames("inline-flex", className)}
        onClick={toggle}
        onKeyDown={handleTriggerKeyDown}
      >
        {trigger}
      </div>
      {open &&
        pos &&
        createPortal(
          <div
            className={classNames(
              "fixed z-50 min-w-[12rem] rounded-xl border border-gray-200/60 bg-white py-1 shadow-lg shadow-gray-500/15 dark:border-gray-700/60 dark:bg-gray-900",
              align === "right" ? "origin-top-right" : "origin-top-left",
              "dropdown-animate-open",
            )}
            style={{
              left: align === "right" ? pos.x + pos.width - menuWidth : pos.x,
              top: pos.y + pos.height + 4,
            }}
            onKeyDown={handleMenuKeyDown}
            tabIndex={-1}
            ref={(el) => {
              menuRef.current = el;
              el?.focus();
            }}
          >
            {items.map((item, i) => {
              if (item.separator) {
                return (
                  <div
                    key={item.id}
                    className="dropdown-item-animate my-1 border-t border-gray-200 dark:border-gray-700"
                    style={{ animationDelay: `${i * 30}ms` }}
                  />
                );
              }

              const isFocused = focusedIndex === i;
              const Icon = item.icon;

              return (
                <button
                  key={item.id}
                  type="button"
                  style={{ animationDelay: `${i * 30}ms` }}
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
                    "dropdown-item-animate flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm outline-none transition duration-100",
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

export default Dropdown;
