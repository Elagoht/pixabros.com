import { IconSearch } from "@tabler/icons-react";
import classNames from "classnames";
import {
  type FC,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

interface CommandItem {
  id: string;
  label: string;
  description?: string;
  icon?: IconElement;
  onSelect: () => void;
}

interface CommandGroup {
  heading: string;
  items: CommandItem[];
}

interface CommandPaletteContentProps {
  onClose: () => void;
  groups: CommandGroup[];
  placeholder: string;
  emptyState: ReactNode;
}

const CommandPaletteContent: FC<CommandPaletteContentProps> = ({
  onClose,
  groups,
  placeholder,
  emptyState,
}) => {
  const [query, setQuery] = useState("");
  const [activeIndex, setActiveIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const filteredGroups = useMemo(
    () =>
      groups
        .map((g) => ({
          heading: g.heading,
          items: g.items.filter(
            (item) =>
              item.label.toLowerCase().includes(query.toLowerCase()) ||
              item.description?.toLowerCase().includes(query.toLowerCase()),
          ),
        }))
        .filter((g) => g.items.length > 0),
    [groups, query],
  );

  const flatItems = useMemo(
    () => filteredGroups.flatMap((g) => g.items),
    [filteredGroups],
  );

  useEffect(() => {
    const id = setTimeout(() => inputRef.current?.focus(), 50);
    return () => clearTimeout(id);
  }, []);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveIndex((i) => Math.min(i + 1, flatItems.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        flatItems[activeIndex]?.onSelect();
        onClose();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [activeIndex, flatItems, onClose]);

  useEffect(() => {
    const el = listRef.current?.querySelector("[data-active]");
    el?.scrollIntoView({ block: "nearest" });
  }, []);

  const handleItemClick = useCallback(
    (item: CommandItem) => {
      item.onSelect();
      onClose();
    },
    [onClose],
  );

  return (
    <div
      className="w-full max-w-lg overflow-hidden rounded-xl border border-gray-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-900"
      onClick={(e) => e.stopPropagation()}
    >
      <div className="flex items-center border-b border-gray-200 px-4 dark:border-gray-700">
        <IconSearch size={18} className="shrink-0 text-gray-400" />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setActiveIndex(0);
          }}
          placeholder={placeholder}
          className="flex-1 border-0 bg-transparent px-3 py-3.5 text-sm text-gray-900 outline-none placeholder:text-gray-400 dark:text-gray-50 dark:placeholder:text-gray-500"
        />
        <kbd className="inline-flex items-center rounded border border-b-2 border-gray-300 bg-gray-100 px-1.5 py-0.5 font-mono text-[10px] font-medium text-gray-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-400">
          ESC
        </kbd>
      </div>

      <div ref={listRef} className="max-h-72 overflow-y-auto p-2">
        {filteredGroups.length > 0 ? (
          filteredGroups.map((group) => (
            <div key={group.heading}>
              <div className="px-2 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                {group.heading}
              </div>
              {group.items.map((item) => {
                const globalIdx = flatItems.indexOf(item);
                const isActive = globalIdx === activeIndex;
                const Icon = item.icon;

                return (
                  <button
                    key={item.id}
                    type="button"
                    data-active={isActive ? "" : undefined}
                    onClick={() => handleItemClick(item)}
                    className={classNames(
                      "flex w-full items-center gap-3 rounded-md px-2 py-2 text-left text-sm transition-colors",
                      isActive
                        ? "bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300"
                        : "text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800",
                    )}
                  >
                    {Icon && (
                      <Icon
                        size={18}
                        className="shrink-0 text-gray-400 dark:text-gray-500"
                      />
                    )}
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-medium">{item.label}</div>
                      {item.description && (
                        <div className="truncate text-xs text-gray-400 dark:text-gray-500">
                          {item.description}
                        </div>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>
          ))
        ) : (
          <div className="px-2 py-6 text-center text-sm text-gray-400 dark:text-gray-500">
            {emptyState}
          </div>
        )}
      </div>
    </div>
  );
};

export default CommandPaletteContent;
