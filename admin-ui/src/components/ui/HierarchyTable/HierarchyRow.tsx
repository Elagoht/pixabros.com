import { useDraggable, useDroppable } from "@dnd-kit/core";
import { IconChevronDown, IconGripVertical } from "@tabler/icons-react";
import classNames from "classnames";
import { useCallback } from "react";

interface HierarchyRowProps<T> {
  node: HierarchyNode<T> & { depth: number; hasChildren: boolean };
  isExpanded: boolean;
  isDropTarget: boolean;
  dropPosition: "before" | "after" | "child" | null;
  onToggle: () => void;
  renderRow: (node: HierarchyNode<T>) => React.ReactNode;
  renderActions?: (node: HierarchyNode<T>) => React.ReactNode;
}

const HierarchyRow = <T,>({
  node,
  isExpanded,
  isDropTarget,
  dropPosition,
  onToggle,
  renderRow,
  renderActions,
}: HierarchyRowProps<T>) => {
  const {
    attributes,
    listeners,
    setNodeRef: setDraggableRef,
    transform,
    isDragging,
  } = useDraggable({ id: node.id });

  const { setNodeRef: setDroppableRef } = useDroppable({ id: node.id });

  const ref = useCallback(
    (el: HTMLElement | null) => {
      setDraggableRef(el);
      setDroppableRef(el);
    },
    [setDraggableRef, setDroppableRef],
  );

  const style: React.CSSProperties | undefined = transform
    ? {
        transform: `translate3d(${transform.x}px, ${transform.y}px, 0)`,
        zIndex: isDragging ? 50 : undefined,
        opacity: isDragging ? 0.5 : undefined,
        position: isDragging ? ("relative" as const) : undefined,
      }
    : undefined;

  return (
    <li
      ref={ref}
      data-row-id={node.id}
      style={style}
      className={classNames(
        "group relative flex items-center gap-1.5 px-4 py-2.5 transition-all duration-200",
        !isDragging && "hover:bg-gray-50 dark:hover:bg-gray-800",
      )}
    >
      {isDropTarget && dropPosition === "before" && (
        <div className="absolute inset-x-0 top-0 z-10 h-0.5 bg-primary-500 shadow-sm shadow-primary-500/30" />
      )}
      {isDropTarget && dropPosition === "child" && (
        <div className="absolute bottom-1 left-0 top-1 z-10 w-0.5 rounded-full bg-primary-500 shadow-sm shadow-primary-500/30" />
      )}
      {isDropTarget && dropPosition === "after" && (
        <div className="absolute inset-x-0 bottom-0 z-10 h-0.5 bg-primary-500 shadow-sm shadow-primary-500/30" />
      )}

      <div style={{ width: node.depth * 20, flexShrink: 0 }} />

      <button
        type="button"
        {...listeners}
        {...attributes}
        className="flex shrink-0 cursor-grab rounded-lg p-1 text-gray-400 transition-all hover:text-gray-600 active:cursor-grabbing hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-800"
        tabIndex={-1}
        aria-label="Drag to reorder"
      >
        <IconGripVertical size={16} />
      </button>

      {node.hasChildren && (
        <button
          type="button"
          onClick={onToggle}
          className={classNames(
            "flex shrink-0 rounded-lg p-1 text-gray-400 transition-all duration-200 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-800",
            isExpanded ? "rotate-0" : "-rotate-90",
          )}
          aria-label={isExpanded ? "Collapse" : "Expand"}
        >
          <IconChevronDown size={16} />
        </button>
      )}
      {!node.hasChildren && <div className="w-[24px] shrink-0" />}

      <div className="min-w-0 flex-1 truncate text-gray-700 dark:text-gray-300">
        {renderRow(node)}
      </div>

      {renderActions && (
        <div className="flex shrink-0 items-center gap-1">
          {renderActions(node)}
        </div>
      )}
    </li>
  );
};

export default HierarchyRow;
