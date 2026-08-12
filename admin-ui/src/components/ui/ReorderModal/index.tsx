import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { type FC, useEffect, useState } from "react";
import { useI18n } from "@/lib/stores/i18n";
import Button from "../Button";
import Modal from "../Modal";
import SortableRow from "./SortableRow";

interface ReorderModalItem {
  id: string;
  label: string;
}

interface ReorderModalProps {
  open: boolean;
  items: ReorderModalItem[];
  title: string;
  help: string;
  isSaving: boolean;
  onClose: () => void;
  onSave: (orderedIds: string[]) => void;
}

const moveItem = <T,>(items: T[], from: number, to: number): T[] => {
  const next = [...items];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next;
};

// A drag-to-order list in a modal. Callers pass {id, label} rows and get back
// the complete ordered id list, which is what the reorder endpoints take.
const ReorderModal: FC<ReorderModalProps> = ({
  open,
  items,
  title,
  help,
  isSaving,
  onClose,
  onSave,
}) => {
  const { t } = useI18n();
  const [ordered, setOrdered] = useState<ReorderModalItem[]>(items);

  // Reopening starts from the server's current order, not whatever
  // half-finished arrangement was abandoned last time.
  useEffect(() => {
    if (open) {
      setOrdered(items);
    }
  }, [open, items]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) {
      return;
    }
    const from = ordered.findIndex((item) => item.id === active.id);
    const to = ordered.findIndex((item) => item.id === over.id);
    if (from === -1 || to === -1) {
      return;
    }
    setOrdered(moveItem(ordered, from, to));
  };

  return (
    <Modal open={open} onClose={onClose} className="w-full max-w-lg">
      <Modal.Header onClose={onClose}>
        <h2 className="text-base font-semibold text-gray-800 dark:text-gray-100">
          {title}
        </h2>
      </Modal.Header>

      <Modal.Body className="space-y-3">
        <p className="text-xs text-gray-500 dark:text-gray-400">{help}</p>

        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <ul className="max-h-80 space-y-2 overflow-y-auto pr-1">
            {ordered.map((item, index) => (
              <SortableRow
                key={item.id}
                id={item.id}
                index={index}
                label={item.label}
              />
            ))}
          </ul>
        </DndContext>
      </Modal.Body>

      <Modal.Footer className="justify-end gap-2">
        <Button variant="outline" onClick={onClose} disabled={isSaving}>
          {t("common.cancel")}
        </Button>
        <Button
          variant="default"
          disabled={isSaving}
          onClick={() => onSave(ordered.map((item) => item.id))}
        >
          {isSaving ? t("common.loading") : t("common.save")}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default ReorderModal;
