import { IconX } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, type ReactNode, useCallback, useEffect } from "react";
import { createPortal } from "react-dom";

interface ModalProps {
  open: boolean;
  onClose?: () => void;
  persistent?: boolean;
  children: ReactNode;
  className?: string;
}

interface ModalSectionProps {
  children: ReactNode;
  className?: string;
}

interface ModalHeaderProps extends ModalSectionProps {
  onClose?: () => void;
}

const Modal: FC<ModalProps> & {
  Header: FC<ModalHeaderProps>;
  Body: FC<ModalSectionProps>;
  Footer: FC<ModalSectionProps>;
} = ({ open, onClose, persistent = false, children, className }) => {
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

  return createPortal(
    <div
      className={classNames(
        "fixed inset-0 z-50 flex items-end justify-center md:items-center",
        "duration-250 transition-all ease-out",
        open ? "bg-black/40" : "pointer-events-none bg-black/0",
      )}
      onClick={handleClose}
    >
      <div
        className={classNames(
          "duration-250 w-full rounded-t-xl bg-white shadow-xl shadow-gray-500/20 transition-all ease-out dark:bg-gray-900",
          "md:max-w-lg md:rounded-xl",
          open
            ? "translate-y-0 opacity-100 md:scale-100"
            : "translate-y-full opacity-0 md:translate-y-4 md:scale-95",
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

const Header: FC<ModalHeaderProps> = ({ children, onClose, className }) => (
  <div
    className={classNames(
      "flex items-center justify-between border-b border-gray-200 px-5 py-4 dark:border-gray-700",
      className,
    )}
  >
    <div className="flex-1">{children}</div>
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

const Body: FC<ModalSectionProps> = ({ children, className }) => (
  <div
    className={classNames("max-h-[70vh] overflow-y-auto px-5 py-4", className)}
  >
    {children}
  </div>
);

const Footer: FC<ModalSectionProps> = ({ children, className }) => (
  <div
    className={classNames(
      "flex items-center justify-end gap-2 border-t border-gray-200 px-5 py-4 dark:border-gray-700",
      className,
    )}
  >
    {children}
  </div>
);

Modal.Header = Header;
Modal.Body = Body;
Modal.Footer = Footer;

export default Modal;
