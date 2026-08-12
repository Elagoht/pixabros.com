import { IconCheck } from "@tabler/icons-react";
import classNames from "classnames";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { filterSelectBase } from "@/utilities/constants";

interface MultiselectDropdownProps {
  options: string[];
  labels?: Record<string, string>;
  selected: string[];
  onChange: (values: string[]) => void;
}

const MultiselectDropdown = ({
  options,
  labels,
  selected,
  onChange,
}: MultiselectDropdownProps) => {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const rect = btnRef.current?.getBoundingClientRect();
    if (rect) {
      setPos({ top: rect.bottom + 2, left: rect.left });
    }
    const handler = (e: MouseEvent) => {
      if (
        menuRef.current &&
        !menuRef.current.contains(e.target as Node) &&
        btnRef.current &&
        !btnRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
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
      if (e.key === "Escape") {
        setOpen(false);
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open]);

  return (
    <div className="relative">
      <button
        ref={btnRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={classNames(
          filterSelectBase,
          "flex w-full items-center gap-1 text-left",
        )}
      >
        <span className="flex-1 truncate">
          {selected.length > 0 ? `${selected.length} seçili` : "Seçiniz..."}
        </span>
        <svg
          className={classNames(
            "h-3 w-3 shrink-0 text-gray-400 transition-transform",
            open && "rotate-180",
          )}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path d="M6 9l6 6 6-6" />
        </svg>
      </button>

      {open &&
        createPortal(
          <div
            ref={menuRef}
            className="fixed z-50 min-w-[10rem] rounded-md border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-900"
            style={{ top: pos.top, left: pos.left }}
          >
            {options.map((opt) => {
              const isSelected = selected.includes(opt);
              return (
                <button
                  key={opt}
                  type="button"
                  onClick={() => {
                    onChange(
                      isSelected
                        ? selected.filter((s) => s !== opt)
                        : [...selected, opt],
                    );
                  }}
                  className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800"
                >
                  <span className="relative inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center">
                    <input
                      type="checkbox"
                      checked={isSelected}
                      readOnly
                      className="h-3.5 w-3.5 appearance-none rounded border border-gray-300 bg-white checked:border-primary-500 checked:bg-primary-500 dark:border-gray-600 dark:bg-gray-900 dark:checked:bg-primary-500"
                    />
                    {isSelected && (
                      <IconCheck
                        size={10}
                        strokeWidth={3}
                        className="pointer-events-none absolute text-white"
                      />
                    )}
                  </span>
                  {labels?.[opt] ?? opt}
                </button>
              );
            })}
          </div>,
          document.body,
        )}
    </div>
  );
};

export default MultiselectDropdown;
