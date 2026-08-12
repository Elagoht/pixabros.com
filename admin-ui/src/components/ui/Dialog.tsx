import type { FC } from "react";
import Button from "./Button";
import Modal from "./Modal";

type ConfirmVariant = "default" | "secondary" | "destructive" | "outline";

interface DialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  titleIcon?: IconElement;
  description?: string;
  onConfirm: () => void;
  onCancel?: () => void;
  confirmLabel?: string;
  cancelLabel?: string;
  confirmVariant?: ConfirmVariant;
}

const Dialog: FC<DialogProps> = ({
  open,
  onClose,
  title,
  titleIcon: TitleIcon,
  description,
  onConfirm,
  onCancel,
  confirmLabel = "Yes",
  cancelLabel = "No",
  confirmVariant = "default",
}) => {
  const handleCancel = () => {
    onCancel?.();
    onClose();
  };

  return (
    <Modal open={open} onClose={onClose}>
      <Modal.Header>
        <div className="flex items-center gap-2">
          {TitleIcon && (
            <TitleIcon
              size={18}
              className="shrink-0 text-gray-500 dark:text-gray-400"
            />
          )}
          <span className="text-sm font-semibold text-gray-900 dark:text-gray-50">
            {title}
          </span>
        </div>
      </Modal.Header>

      {description && (
        <Modal.Body>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            {description}
          </p>
        </Modal.Body>
      )}

      <Modal.Footer>
        <Button variant="outline" size="sm" onClick={handleCancel}>
          {cancelLabel}
        </Button>
        <Button variant={confirmVariant} size="sm" onClick={onConfirm}>
          {confirmLabel}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default Dialog;
