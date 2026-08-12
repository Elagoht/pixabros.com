import { IconX } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, type ReactNode, useCallback, useEffect } from "react";
import { createPortal } from "react-dom";

type DrawerPosition = "left" | "right";

interface DrawerProps {
  open: boolean;
  onClose?: () => void;
  persistent?: boolean;
  position?: DrawerPosition;
  children: ReactNode;
  className?: string;
}

interface DrawerSectionProps {
  children: ReactNode;
  className?: string;
}

interface DrawerHeaderProps extends DrawerSectionProps {
  onClose?: () => void;
}

const Drawer: FC<DrawerProps> & {
  Header: FC<DrawerHeaderProps>;
  Body: FC<DrawerSectionProps>;
  Footer: FC<DrawerSectionProps>;
} = ({
  open,
  onClose,
  persistent = false,
  position = "right",
  children,
  className,
}) => {
  const handleClose = useCallback(() => {
    if (!persistent) {
      onClose?.();
    }
  }, [persistent, onClose]);

  useEffect(() => {
    if (!open || persistent) {
      return;
    }
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        handleClose();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, persistent, handleClose]);

  useEffect(() => {
    if (open) {
      const prev = document.body.style.overflow;
      document.body.style.overflow = "hidden";
      return () => {
        document.body.style.overflow = prev;
      };
    }
  }, [open]);

  const slideFrom =
    position === "left"
      ? open
        ? "translate-x-0"
        : "-translate-x-full"
      : open
        ? "translate-x-0"
        : "translate-x-full";

  return createPortal(
    <div
      className={classNames(
        "fixed inset-0 z-50 flex",
        "transition-all duration-700 ease-out",
        position === "left" ? "justify-start" : "justify-end",
        open ? "bg-black/40" : "pointer-events-none bg-black/0",
      )}
      onClick={handleClose}
    >
      <div
        className={classNames(
          "flex h-full w-full max-w-sm flex-col bg-white shadow-xl transition-all duration-700 ease-out dark:bg-gray-900",
          "md:max-w-md",
          slideFrom,
          open ? "" : "pointer-events-none",
          className,
        )}
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>,
    document.body,
  );
};

const Header: FC<DrawerHeaderProps> = ({ children, onClose, className }) => (
  <div
    className={classNames(
      "flex items-center justify-between border-b border-gray-200 px-5 py-4 dark:border-gray-700",
      className,
    )}
  >
    <div className="flex-1 text-sm font-semibold text-gray-900 dark:text-gray-50">
      {children}
    </div>
    {onClose && (
      <button
        type="button"
        onClick={onClose}
        className="ml-3 rounded-md p-1 text-gray-400 transition hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
      >
        <IconX size={18} />
      </button>
    )}
  </div>
);

const Body: FC<DrawerSectionProps> = ({ children, className }) => (
  <div className={classNames("flex-1 overflow-y-auto px-5 py-4", className)}>
    {children}
  </div>
);

const Footer: FC<DrawerSectionProps> = ({ children, className }) => (
  <div
    className={classNames(
      "flex items-center justify-end gap-2 border-t border-gray-200 px-5 py-4 dark:border-gray-700",
      className,
    )}
  >
    {children}
  </div>
);

Drawer.Header = Header;
Drawer.Body = Body;
Drawer.Footer = Footer;

export default Drawer;
