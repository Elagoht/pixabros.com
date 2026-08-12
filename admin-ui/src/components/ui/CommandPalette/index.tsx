import type { FC, ReactNode } from "react";
import { createPortal } from "react-dom";
import CommandPaletteContent from "./CommandPaletteContent";

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

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
  groups: CommandGroup[];
  placeholder?: string;
  emptyState?: ReactNode;
}

const CommandPalette: FC<CommandPaletteProps> = ({
  open,
  onClose,
  groups,
  placeholder = "Type a command or search...",
  emptyState = "No results found.",
}) => {
  if (!open) {
    return null;
  }

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 pt-[15vh]"
      onClick={onClose}
    >
      <CommandPaletteContent
        onClose={onClose}
        groups={groups}
        placeholder={placeholder}
        emptyState={emptyState}
      />
    </div>,
    document.body,
  );
};

export default CommandPalette;
