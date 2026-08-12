import { useDraggable, useDroppable } from "@dnd-kit/core";
import { IconGripVertical } from "@tabler/icons-react";
import classNames from "classnames";
import { type CSSProperties, type FC, useCallback } from "react";

interface SortableRowProps {
  id: string;
  index: number;
  label: string;
}

const SortableRow: FC<SortableRowProps> = ({ id, index, label }) => {
  const {
    attributes,
    listeners,
    setNodeRef: setDraggableRef,
    transform,
    isDragging,
  } = useDraggable({ id });
  const { setNodeRef: setDroppableRef, isOver } = useDroppable({ id });

  const ref = useCallback(
    (el: HTMLElement | null) => {
      setDraggableRef(el);
      setDroppableRef(el);
    },
    [setDraggableRef, setDroppableRef],
  );

  const style: CSSProperties | undefined = transform
    ? {
        transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
        zIndex: isDragging ? 50 : undefined,
        opacity: isDragging ? 0.5 : undefined,
        position: isDragging ? "relative" : undefined,
      }
    : undefined;

  return (
    <li
      ref={ref}
      style={style}
      className={classNames(
        "flex items-center gap-3 rounded-lg border bg-white px-3 py-2 dark:bg-gray-900",
        isOver && !isDragging
          ? "border-primary-500 ring-2 ring-primary-500/30"
          : "border-gray-200 dark:border-gray-700",
      )}
    >
      <button
        type="button"
        className="cursor-grab touch-none text-gray-400 hover:text-gray-600 active:cursor-grabbing dark:text-gray-500 dark:hover:text-gray-300"
        {...attributes}
        {...listeners}
      >
        <IconGripVertical size={16} />
      </button>
      <span className="w-6 shrink-0 text-xs tabular-nums text-gray-400 dark:text-gray-500">
        {index + 1}
      </span>
      <span className="min-w-0 flex-1 truncate text-sm text-gray-800 dark:text-gray-100">
        {label}
      </span>
    </li>
  );
};

export default SortableRow;
