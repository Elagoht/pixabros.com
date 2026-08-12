import {
  DndContext,
  type DragEndEvent,
  type DragMoveEvent,
  DragOverlay,
  type DragStartEvent,
  PointerSensor,
  pointerWithin,
  useDroppable,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { IconGripVertical, IconHierarchy2 } from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { useCallback, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import {
  applyDrop,
  detectCycle,
  getDepth,
  hasChildren,
} from "@/utilities/hierarchy-table";
import HierarchyRow from "./HierarchyRow";

const HierarchyTable = <T,>(props: HierarchyTableProps<T>) => {
  const {
    name,
    renderRow,
    renderActions,
    saveButtonText = "Save",
    emptyState,
  } = props;

  const [field, , helpers] = useField<HierarchyNode<T>[]>(name);
  const formikValues = field.value || [];

  const [items, setItems] = useState<HierarchyNode<T>[]>(() =>
    formikValues.length > 0
      ? formikValues.map((n) => ({ ...n, data: { ...n.data } }))
      : [],
  );
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => {
    return new Set(formikValues.map((n) => n.id));
  });
  const [hasChanges, setHasChanges] = useState(false);

  const [activeId, setActiveId] = useState<string | null>(null);
  const [dropIndicator, setDropIndicator] = useState<{
    targetId: string;
    position: "before" | "after" | "child";
  } | null>(null);

  const dragStartPointer = useRef({ x: 0, y: 0 });

  const nodesWithDepth = useMemo(
    () =>
      items.map((n) => ({
        id: n.id,
        parentId: n.parentId,
        data: n.data,
        depth: getDepth(items, n.id),
        hasChildren: hasChildren(items, n.id),
      })),
    [items],
  );

  const visibleNodes = useMemo(() => {
    const result: Array<
      HierarchyNode<T> & { depth: number; hasChildren: boolean }
    > = [];

    for (const node of nodesWithDepth) {
      let hidden = false;
      let parentId = node.parentId;
      const ancestors = new Set<string>();
      while (parentId && !hidden) {
        if (ancestors.has(parentId)) {
          break;
        }
        ancestors.add(parentId);
        if (!expandedIds.has(parentId)) {
          hidden = true;
        }
        const parent = nodesWithDepth.find((n) => n.id === parentId);
        parentId = parent?.parentId ?? null;
      }

      if (!hidden) {
        result.push(node);
      }
    }

    return result;
  }, [nodesWithDepth, expandedIds]);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: { distance: 5 },
    }),
  );

  const handleDragStart = useCallback((event: DragStartEvent) => {
    const id = String(event.active.id);
    setActiveId(id);

    const ev = event.activatorEvent;
    if (ev instanceof MouseEvent) {
      dragStartPointer.current = { x: ev.clientX, y: ev.clientY };
    }
  }, []);

  const lastDropIndicator = useRef<{
    targetId: string;
    position: string;
  } | null>(null);

  const handleDragMove = useCallback(
    (event: DragMoveEvent) => {
      const { over } = event;

      if (!(over && activeId) || over.id === activeId) {
        setDropIndicator(null);
        lastDropIndicator.current = null;
        return;
      }

      const targetId = String(over.id);

      const pointerY = dragStartPointer.current.y + event.delta.y;

      const el = document.querySelector(`[data-row-id="${targetId}"]`);
      if (!el) {
        return;
      }

      const rect = el.getBoundingClientRect();
      const relativeY = (pointerY - rect.top) / rect.height;

      let position: "before" | "after" | "child";
      if (relativeY < 0.25) {
        position = "before";
      } else if (relativeY > 0.75) {
        position = "after";
      } else {
        position = "child";
      }

      if (
        lastDropIndicator.current?.targetId === targetId &&
        lastDropIndicator.current?.position === position
      ) {
        return;
      }

      lastDropIndicator.current = { targetId, position };
      setDropIndicator({ targetId, position });
    },
    [activeId],
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;

      if (over && active.id !== over.id) {
        const draggedId = String(active.id);

        if (over.id === "__top-level__") {
          const newItems = applyDrop(items, draggedId, null, "child");
          if (newItems !== items) {
            setItems(newItems);
            setHasChanges(true);
          }
        } else {
          const targetId = String(over.id);

          const pointerY = dragStartPointer.current.y + event.delta.y;
          const el = document.querySelector(`[data-row-id="${targetId}"]`);

          if (el) {
            const rect = el.getBoundingClientRect();
            const relativeY = (pointerY - rect.top) / rect.height;

            let position: "before" | "after" | "child";
            if (relativeY < 0.25) {
              position = "before";
            } else if (relativeY > 0.75) {
              position = "after";
            } else {
              position = "child";
            }

            if (position === "child") {
              if (detectCycle(items, draggedId, targetId)) {
                toast.error("Cannot create cyclic parent reference");
                setActiveId(null);
                setDropIndicator(null);
                return;
              }
            }

            const newItems = applyDrop(items, draggedId, targetId, position);
            if (newItems !== items) {
              setItems(newItems);
              setHasChanges(true);
            }
          }
        }
      }

      setActiveId(null);
      setDropIndicator(null);
      lastDropIndicator.current = null;
    },
    [items],
  );

  const { setNodeRef: topLevelRef, isOver: isOverTopLevel } = useDroppable({
    id: "__top-level__",
  });

  const handleSave = useCallback(() => {
    helpers.setValue(items);
    setHasChanges(false);
    toast.success("Hierarchy saved");
  }, [items, helpers]);

  const activeNode = activeId ? items.find((n) => n.id === activeId) : null;

  const toggleExpand = useCallback((id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const itemsExist = items.length > 0;

  return (
    <div className="rounded-xl border border-gray-200/60 bg-white dark:border-gray-700/60 dark:bg-gray-900">
      <div className="flex items-center gap-2 border-b border-gray-200/60 px-4 py-3 dark:border-gray-700/60">
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          Hierarchy
        </span>
        {hasChanges && (
          <span className="ml-auto text-xs text-amber-600 dark:text-amber-500">
            Unsaved changes
          </span>
        )}
      </div>

      <DndContext
        sensors={sensors}
        collisionDetection={pointerWithin}
        onDragStart={handleDragStart}
        onDragMove={handleDragMove}
        onDragEnd={handleDragEnd}
      >
        <ul
          ref={topLevelRef}
          className={classNames(
            "divide-y divide-gray-100/60 dark:divide-gray-700/30",
            isOverTopLevel &&
              items.length > 0 &&
              "bg-primary-50 dark:bg-primary-900/15",
          )}
        >
          {visibleNodes.length > 0
            ? visibleNodes.map((node) => (
                <HierarchyRow
                  key={node.id}
                  node={node}
                  isExpanded={expandedIds.has(node.id)}
                  isDropTarget={dropIndicator?.targetId === node.id}
                  dropPosition={
                    dropIndicator?.targetId === node.id
                      ? dropIndicator.position
                      : null
                  }
                  onToggle={() => toggleExpand(node.id)}
                  renderRow={renderRow}
                  renderActions={renderActions}
                />
              ))
            : !itemsExist && (
                <div className="flex flex-col items-center justify-center py-12 text-gray-400 dark:text-gray-500">
                  <IconHierarchy2 size={32} className="mb-2" />
                  <p className="text-sm">{emptyState || "No items"}</p>
                </div>
              )}

          {isOverTopLevel && itemsExist && (
            <div className="flex items-center justify-center border-t-2 border-primary-500 bg-primary-50 px-4 py-3 dark:bg-primary-900/15">
              <span className="text-xs font-medium text-primary-600 dark:text-primary-400">
                Drop here to make top-level
              </span>
            </div>
          )}
        </ul>

        <DragOverlay>
          {activeNode ? (
            <div className="flex items-center gap-2 rounded-lg border border-gray-200/60 bg-white px-4 py-2.5 shadow-xl shadow-gray-500/20 dark:border-gray-700/60 dark:bg-gray-900">
              <IconGripVertical
                size={16}
                className="shrink-0 text-gray-400 dark:text-gray-500"
              />
              <div className="min-w-0 flex-1 truncate">
                {renderRow(activeNode)}
              </div>
              {renderActions && (
                <div className="flex shrink-0 items-center gap-1 opacity-50">
                  {renderActions(activeNode)}
                </div>
              )}
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>

      <div className="flex items-center justify-end border-t border-gray-200/60 px-4 py-3 dark:border-gray-700/60">
        <button
          type="button"
          onClick={handleSave}
          disabled={!hasChanges}
          className={classNames(
            "inline-flex items-center justify-center rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
            "ring-offset-white dark:ring-offset-gray-950",
            hasChanges
              ? "bg-primary-500 text-white shadow-md shadow-primary-500/30 hover:bg-primary-400 hover:shadow-lg hover:shadow-primary-500/40 hover:ring-2 hover:ring-primary-400/50 hover:ring-offset-2 focus-visible:ring-primary-500"
              : "cursor-not-allowed bg-gray-100 text-gray-400 dark:bg-gray-800 dark:text-gray-500",
          )}
        >
          {saveButtonText}
        </button>
      </div>
    </div>
  );
};

export default HierarchyTable;
