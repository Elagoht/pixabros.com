import { useDraggable, useDroppable } from "@dnd-kit/core";
import { IconGripVertical, IconTrash, IconWorld } from "@tabler/icons-react";
import classNames from "classnames";
import { type CSSProperties, type FC, useCallback } from "react";
import Button from "../Button";

interface LinkRowProps {
  id: string;
  label: string;
  url: string;
  labelPlaceholder: string;
  urlPlaceholder: string;
  removeLabel: string;
  labelError?: string;
  urlError?: string;
  draggable: boolean;
  onChange: (patch: { label?: string; url?: string }) => void;
  onBlur: () => void;
  onRemove: () => void;
}

const inputClasses = (hasError: boolean) =>
  classNames(
    "w-full rounded-lg border bg-gray-50 px-3 py-2 text-sm outline-none transition-all duration-200",
    "text-gray-900 placeholder-gray-400 dark:bg-gray-800/50 dark:text-gray-50 dark:placeholder-gray-500",
    hasError
      ? "border-red-500 focus-visible:ring-2 focus-visible:ring-red-500"
      : "border-gray-200 focus-visible:ring-2 focus-visible:ring-primary-500 dark:border-gray-700",
  );

const LinkRow: FC<LinkRowProps> = ({
  id,
  label,
  url,
  labelPlaceholder,
  urlPlaceholder,
  removeLabel,
  labelError,
  urlError,
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

      <div className="w-1/3 shrink-0">
        <input
          type="text"
          value={label}
          placeholder={labelPlaceholder}
          onChange={(e) => onChange({ label: e.target.value })}
          onBlur={onBlur}
          className={inputClasses(!!labelError)}
        />
        {labelError && (
          <p className="mt-1 text-xs text-red-500">{labelError}</p>
        )}
      </div>

      <div className="min-w-0 flex-1">
        <div className="relative">
          <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-gray-400">
            <IconWorld size={16} />
          </span>
          <input
            type="url"
            value={url}
            placeholder={urlPlaceholder}
            onChange={(e) => onChange({ url: e.target.value })}
            onBlur={onBlur}
            className={classNames(inputClasses(!!urlError), "pl-9")}
          />
        </div>
        {urlError && <p className="mt-1 text-xs text-red-500">{urlError}</p>}
      </div>

      <Button variant="ghost" size="md" title={removeLabel} onClick={onRemove}>
        <IconTrash size={16} />
      </Button>
    </div>
  );
};

export default LinkRow;
