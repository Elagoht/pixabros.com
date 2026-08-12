import { IconX } from "@tabler/icons-react";
import type { FC, ReactNode } from "react";

interface ChipProps {
  children: ReactNode;
  onRemove: () => void;
}

const Chip: FC<ChipProps> = ({ children, onRemove }) => (
  <span className="inline-flex items-center gap-0.5 rounded-full bg-primary-100 px-2 py-1 text-xs font-medium text-primary-700 transition-all duration-200 dark:bg-primary-900/30 dark:text-primary-300">
    <button
      type="button"
      tabIndex={-1}
      onClick={(e) => {
        e.stopPropagation();
        onRemove();
      }}
      className="text-primary-400 hover:text-red-500 transition-colors duration-150"
    >
      <IconX size={10} />
    </button>
    {children}
  </span>
);

export default Chip;
