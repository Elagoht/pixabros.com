import { useDraggable, useDroppable } from "@dnd-kit/core";
import { IconGripVertical, IconTrash } from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import classNames from "classnames";
import { type CSSProperties, type FC, useCallback } from "react";
import { Button, ImagePreview, Skeleton } from "@/components/ui";
import { queryKeys } from "@/lib/query/keys";
import { MediaService } from "@/services/media";

interface SortableScreenshotProps {
  id: string;
  mediaId: string;
  position: number;
  alt: string;
  removeLabel: string;
  isBusy: boolean;
  onRemove: () => void;
}

const SortableScreenshot: FC<SortableScreenshotProps> = ({
  id,
  mediaId,
  position,
  alt,
  removeLabel,
  isBusy,
  onRemove,
}) => {
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.media.detail(mediaId),
    queryFn: () => MediaService.get(mediaId),
  });

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
        "space-y-1.5 rounded-lg border p-2 transition-colors",
        isOver && !isDragging
          ? "border-primary-500 ring-2 ring-primary-500/30"
          : "border-gray-200 dark:border-gray-700",
      )}
    >
      {isLoading ? (
        <Skeleton className="h-20 w-full" variant="rect" />
      ) : data ? (
        <ImagePreview
          src={data.url}
          alt={alt}
          caption={alt}
          className="h-20 object-cover"
        />
      ) : null}

      <div className="flex items-center justify-between">
        <span className="flex items-center gap-1">
          <button
            type="button"
            className="cursor-grab touch-none text-gray-400 hover:text-gray-600 active:cursor-grabbing dark:text-gray-500 dark:hover:text-gray-300"
            {...attributes}
            {...listeners}
          >
            <IconGripVertical size={14} />
          </button>
          <span className="text-[10px] tabular-nums text-gray-400 dark:text-gray-500">
            #{position}
          </span>
        </span>
        <Button
          variant="ghost"
          size="sm"
          title={removeLabel}
          disabled={isBusy}
          onClick={onRemove}
        >
          <IconTrash size={14} />
        </Button>
      </div>
    </li>
  );
};

export default SortableScreenshot;
