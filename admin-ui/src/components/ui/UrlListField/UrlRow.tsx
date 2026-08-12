import { useDraggable, useDroppable } from "@dnd-kit/core";
import { IconGripVertical, IconTrash, IconWorld } from "@tabler/icons-react";
import classNames from "classnames";
import { type CSSProperties, type FC, useCallback } from "react";
import Button from "../Button";

interface UrlRowProps {
  id: string;
  url: string;
  placeholder: string;
  removeLabel: string;
  error?: string;
  draggable: boolean;
  onChange: (value: string) => void;
  onBlur: () => void;
  onRemove: () => void;
}

const UrlRow: FC<UrlRowProps> = ({
  id,
  url,
  placeholder,
  removeLabel,
  error,
  draggable,
  onChange,
  onBlur,
  onRemove,
}) => {
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
    <div
      ref={ref}
      style={style}
      className={classNames(
        "flex items-start gap-2 rounded-lg transition-colors",
        isOver && !isDragging && "ring-2 ring-primary-500/40",
      )}
    >
      {draggable && (
        <button
          type="button"
          className="mt-2 cursor-grab touch-none text-gray-400 hover:text-gray-600 active:cursor-grabbing dark:text-gray-500 dark:hover:text-gray-300"
          {...attributes}
          {...listeners}
        >
          <IconGripVertical size={16} />
        </button>
      )}

      <div className="min-w-0 flex-1">
        <div className="relative">
          <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-gray-400">
            <IconWorld size={16} />
          </span>
          <input
            type="url"
            value={url}
            placeholder={placeholder}
            onChange={(e) => onChange(e.target.value)}
            onBlur={onBlur}
            className={classNames(
              "w-full rounded-lg border bg-gray-50 py-2 pl-9 pr-3 text-sm outline-none transition-all duration-200",
              "text-gray-900 placeholder-gray-400 dark:bg-gray-800/50 dark:text-gray-50 dark:placeholder-gray-500",
              error
                ? "border-red-500 focus-visible:ring-2 focus-visible:ring-red-500"
                : "border-gray-200 focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-gray-700",
            )}
          />
        </div>
        {error && <p className="mt-1 text-xs text-red-500">{error}</p>}
      </div>

      <Button variant="ghost" size="md" title={removeLabel} onClick={onRemove}>
        <IconTrash size={16} />
      </Button>
    </div>
  );
};

export default UrlRow;
