import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { type FC, useEffect, useState } from "react";
import { Button, Modal } from "@/components/ui";
import { useI18n } from "@/lib/stores/i18n";
import SortableRow from "./SortableRow";

interface ReorderGamesModalProps {
  open: boolean;
  games: ResponseGame[];
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

const ReorderGamesModal: FC<ReorderGamesModalProps> = ({
  open,
  games,
  isSaving,
  onClose,
  onSave,
}) => {
  const { t } = useI18n();
  const [ordered, setOrdered] = useState<ResponseGame[]>(games);

  // Reopening the modal must start from the server's current order, not
  // whatever half-finished arrangement was abandoned last time.
  useEffect(() => {
    if (open) {
      setOrdered(games);
    }
  }, [open, games]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) {
      return;
    }
    const from = ordered.findIndex((game) => game.id === active.id);
    const to = ordered.findIndex((game) => game.id === over.id);
    if (from === -1 || to === -1) {
      return;
    }
    setOrdered(moveItem(ordered, from, to));
  };

  return (
    <Modal open={open} onClose={onClose} className="w-full max-w-lg">
      <Modal.Header onClose={onClose}>
        <h2 className="text-base font-semibold text-gray-800 dark:text-gray-100">
          {t("games.reorder.title")}
        </h2>
      </Modal.Header>

      <Modal.Body className="space-y-3">
        <p className="text-xs text-gray-500 dark:text-gray-400">
          {t("games.reorder.help")}
        </p>

        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <ul className="max-h-80 space-y-2 overflow-y-auto pr-1">
            {ordered.map((game, index) => (
              <SortableRow
                key={game.id}
                id={game.id}
                index={index}
                label={game.title}
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
          onClick={() => onSave(ordered.map((game) => game.id))}
        >
          {isSaving ? t("common.loading") : t("common.save")}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default ReorderGamesModal;
