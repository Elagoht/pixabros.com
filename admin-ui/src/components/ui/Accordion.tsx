import { IconChevronDown } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useState } from "react";

interface AccordionItem {
  value: string;
  label: string;
  content: React.ReactNode;
}

interface AccordionProps {
  items: AccordionItem[];
  type?: "single" | "multiple";
  defaultOpen?: string | string[];
  className?: string;
}

const Accordion: FC<AccordionProps> = ({
  items,
  type = "single",
  defaultOpen,
  className,
}) => {
  const [openItems, setOpenItems] = useState<Set<string>>(() => {
    if (!defaultOpen) {
      return new Set();
    }
    const arr = Array.isArray(defaultOpen) ? defaultOpen : [defaultOpen];
    return new Set(arr);
  });

  const toggle = (value: string) => {
    setOpenItems((prev) => {
      const next = new Set(prev);
      if (next.has(value)) {
        next.delete(value);
      } else {
        if (type === "single") {
          next.clear();
        }
        next.add(value);
      }
      return next;
    });
  };

  return (
    <div
      className={classNames(
        "divide-y divide-gray-200/60 rounded-xl border border-gray-200/60 bg-gray-50 shadow-md shadow-gray-500/10 dark:divide-gray-700/60 dark:border-gray-700/60 dark:bg-gray-800/50",
        className,
      )}
    >
      {items.map((item) => {
        const isOpen = openItems.has(item.value);

        return (
          <div key={item.value}>
            <button
              type="button"
              onClick={() => toggle(item.value)}
              className={classNames(
                "flex w-full items-center justify-between px-4 py-3 text-left text-sm font-medium transition-all duration-200",
                "text-gray-700 hover:bg-gray-50/80 hover:text-gray-900 hover:ring-1 hover:ring-gray-200/50 dark:text-gray-300 dark:hover:bg-gray-800/50 dark:hover:text-gray-100 dark:hover:ring-gray-700/50",
              )}
            >
              {item.label}
              <IconChevronDown
                size={16}
                className={classNames(
                  "shrink-0 text-gray-400 transition-transform duration-200",
                  isOpen && "rotate-180",
                )}
              />
            </button>
            <div
              className={classNames(
                "overflow-hidden transition-all duration-200 ease-out",
                isOpen ? "max-h-96 opacity-100" : "max-h-0 opacity-0",
              )}
            >
              <div className="px-4 pb-3 text-sm text-gray-600 dark:text-gray-400">
                {item.content}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default Accordion;
