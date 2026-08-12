import classNames from "classnames";
import { type FC, useState } from "react";

interface TabItem {
  value: string;
  label: string;
  content: React.ReactNode;
}

interface TabsProps {
  items: TabItem[];
  defaultValue?: string;
  value?: string;
  onChange?: (value: string) => void;
  fullWidth?: boolean;
  className?: string;
}

const Tabs: FC<TabsProps> = ({
  items,
  defaultValue,
  value: controlledValue,
  onChange,
  fullWidth = false,
  className,
}) => {
  const [internalValue, setInternalValue] = useState(
    defaultValue ?? items[0]?.value ?? "",
  );
  const activeValue = controlledValue ?? internalValue;

  const handleChange = (v: string) => {
    if (!controlledValue) {
      setInternalValue(v);
    }
    onChange?.(v);
  };

  const activeTab = items.find((t) => t.value === activeValue);

  return (
    <div className={className}>
      <div
        className={classNames(
          "flex",
          fullWidth
            ? "gap-1 rounded-xl border border-gray-300 bg-gray-50/80 p-1 dark:border-gray-600 dark:bg-gray-800/30"
            : "border-b-2 border-gray-300 dark:border-gray-600",
        )}
      >
        {items.map((tab) => {
          const isActive = tab.value === activeValue;
          return (
            <button
              key={tab.value}
              type="button"
              onClick={() => handleChange(tab.value)}
              className={classNames(
                "relative overflow-hidden px-4 py-2.5 text-sm font-medium transition-all duration-200",
                fullWidth ? "flex-1 rounded-lg" : "rounded-t-lg",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950",
                isActive
                  ? "bg-primary-600 text-white hover:bg-primary-500 active:bg-primary-700"
                  : fullWidth
                    ? "text-gray-500 hover:text-gray-700 hover:bg-white/60 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-gray-700/40"
                    : "text-gray-500 hover:text-gray-700 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-gray-800",
              )}
            >
              {tab.label}
            </button>
          );
        })}
      </div>
      {activeTab && <div className="py-3">{activeTab.content}</div>}
    </div>
  );
};

export default Tabs;
