import {
  IconChevronDown,
  IconFolder,
  IconPencil,
  IconPlus,
  IconTrash,
} from "@tabler/icons-react";
import { useVirtualizer } from "@tanstack/react-virtual";
import classNames from "classnames";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { toast } from "sonner";
import { useI18n } from "@/lib/stores/i18n";
import {
  detectCycle,
  getAllDescendantIds,
  getDepth,
  hasChildren,
} from "@/utilities/hierarchy-table";
import Button from "../Button";
import Dialog from "../Dialog";
import Modal from "../Modal";

type ModalState<T> =
  | { type: "edit"; category: HierarchyNode<T> }
  | { type: "create"; initialParentId: string | null }
  | { type: "delete"; category: HierarchyNode<T> }
  | null;

interface CategoryTableProps<T> {
  categories: HierarchyNode<T>[];
  getCategoryLabel: (category: HierarchyNode<T>) => string;
  onSave: (category: HierarchyNode<T>) => void;
  onCreate: (parentId: string | null, data: T) => void;
  onDelete: (categoryId: string) => void;
  renderFormFields: (data: T, onChange: (data: T) => void) => ReactNode;
  initialFormData: T;
  emptyState?: ReactNode;
}

interface CategoryOption {
  label: string;
  value: string;
}

const CategoryTable = <T,>({
  categories,
  getCategoryLabel,
  onSave,
  onCreate,
  onDelete,
  renderFormFields,
  initialFormData,
  emptyState,
}: CategoryTableProps<T>) => {
  const { t } = useI18n();
  const [expandedIds, setExpandedIds] = useState<Set<string>>(
    () => new Set(categories.map((c) => c.id)),
  );

  useEffect(() => {
    setExpandedIds(new Set(categories.map((c) => c.id)));
  }, [categories]);

  const [modalState, setModalState] = useState<ModalState<T>>(null);
  const [formData, setFormData] = useState<T>(initialFormData);
  const [selectedParentId, setSelectedParentId] = useState("");

  const nodesWithDepth = useMemo(
    () =>
      categories.map((n) => ({
        ...n,
        depth: getDepth(categories, n.id),
        hasChildren: hasChildren(categories, n.id),
      })),
    [categories],
  );

  const visibleNodes = useMemo(() => {
    const result: (HierarchyNode<T> & {
      depth: number;
      hasChildren: boolean;
    })[] = [];

    const addChildren = (parentId: string | null) => {
      const children = nodesWithDepth.filter((n) => n.parentId === parentId);
      for (const child of children) {
        result.push(child);
        if (expandedIds.has(child.id)) {
          addChildren(child.id);
        }
      }
    };

    addChildren(null);

    return result;
  }, [nodesWithDepth, expandedIds]);

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

  const open = useCallback(
    (state: ModalState<T>) => {
      if (state?.type === "edit") {
        setFormData({ ...state.category.data });
        setSelectedParentId(state.category.parentId ?? "");
      } else if (state?.type === "create") {
        setFormData({ ...initialFormData });
        setSelectedParentId(state.initialParentId ?? "");
      }
      setModalState(state);
    },
    [initialFormData],
  );

  const close = useCallback(() => {
    setModalState(null);
  }, []);

  const getParentOptions = useCallback(
    (excludeId: string | null): CategoryOption[] => {
      let availableCategories = categories;

      if (excludeId) {
        const descendants = getAllDescendantIds(categories, excludeId);
        descendants.add(excludeId);
        availableCategories = categories.filter((c) => !descendants.has(c.id));
      }

      const options: CategoryOption[] = [
        { label: t("categories.table.topLevel"), value: "" },
      ];

      for (const c of availableCategories) {
        options.push({
          label: `${"\u00A0\u00A0\u00A0\u00A0".repeat(
            getDepth(categories, c.id),
          )}${getCategoryLabel(c)}`,
          value: c.id,
        });
      }

      return options;
    },
    [categories, getCategoryLabel, t],
  );

  const handleEditSave = useCallback(() => {
    if (!modalState || modalState.type !== "edit") {
      return;
    }
    const parentId = selectedParentId || null;

    if (parentId && detectCycle(categories, modalState.category.id, parentId)) {
      toast.error(t("categories.table.cycleError"));
      return;
    }

    onSave({
      ...modalState.category,
      parentId,
      data: formData,
    });
    close();
  }, [modalState, selectedParentId, formData, categories, onSave, close, t]);

  const handleCreateSave = useCallback(() => {
    const parentId = selectedParentId || null;
    onCreate(parentId, formData);
    close();
  }, [selectedParentId, formData, onCreate, close]);

  const handleDelete = useCallback(() => {
    if (!modalState || modalState.type !== "delete") {
      return;
    }
    onDelete(modalState.category.id);
    close();
  }, [modalState, onDelete, close]);

  const selectClasses = classNames(
    "w-full appearance-none rounded-md border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 shadow-inner",
    "transition duration-150 ease-out",
    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2",
    "dark:border-gray-700 dark:bg-gray-900 dark:text-gray-50 dark:focus-visible:ring-offset-gray-950",
  );

  const deleteTarget =
    modalState?.type === "delete" ? modalState.category : null;
  const hasSubcategories =
    deleteTarget && hasChildren(categories, deleteTarget.id);

  const scrollRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: visibleNodes.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 42,
    overscan: 10,
  });

  return (
    <div className="flex flex-col rounded-lg border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-950 max-h-[calc(100vh-196px)]">
      <div className="flex shrink-0 items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
        <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">
          {t("categories.table.header")}
        </span>
        <Button
          variant="outline"
          size="sm"
          leftIcon={IconPlus}
          onClick={() => open({ type: "create", initialParentId: null })}
        >
          {t("common.create")}
        </Button>
      </div>

      {visibleNodes.length > 0 ? (
        <div ref={scrollRef} className="flex-1 overflow-auto">
          <div
            className="relative divide-y divide-gray-100 dark:divide-gray-800"
            style={{ height: virtualizer.getTotalSize() }}
          >
            {virtualizer.getVirtualItems().map((virtualItem) => {
              const node = visibleNodes[virtualItem.index];

              return (
                <div
                  key={node.id}
                  className={classNames(
                    "absolute left-0 top-0 flex w-full items-center gap-1.5 px-4 py-2.5 group",
                    node.hasChildren
                      ? "bg-gray-50 dark:bg-gray-900/60"
                      : "hover:bg-gray-50 dark:hover:bg-gray-900",
                  )}
                  style={{
                    height: 42,
                    transform: `translateY(${virtualItem.start}px)`,
                  }}
                >
                  <div style={{ width: node.depth * 20, flexShrink: 0 }} />

                  {node.hasChildren ? (
                    <button
                      type="button"
                      onClick={() => toggleExpand(node.id)}
                      className={classNames(
                        "flex shrink-0 rounded p-0.5 text-gray-400 transition-transform hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300",
                        expandedIds.has(node.id) ? "rotate-0" : "-rotate-90",
                      )}
                    >
                      <IconChevronDown size={16} />
                    </button>
                  ) : (
                    <div className="w-6 shrink-0" />
                  )}

                  <div className="min-w-0 flex-1 truncate text-sm text-gray-700 dark:text-gray-300">
                    {getCategoryLabel(node)}
                  </div>

                  <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                    <Button
                      variant="ghost"
                      size="sm"
                      leftIcon={IconPlus}
                      onClick={() =>
                        open({ type: "create", initialParentId: node.id })
                      }
                    >
                      {t("common.create")}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      leftIcon={IconPencil}
                      onClick={() => open({ type: "edit", category: node })}
                    >
                      {t("common.edit")}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      leftIcon={IconTrash}
                      onClick={() => open({ type: "delete", category: node })}
                    >
                      {t("common.delete")}
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center py-12 text-gray-400 dark:text-gray-500">
          <IconFolder size={32} className="mb-2" />
          <p className="text-sm">
            {emptyState ?? t("categories.table.noCategories")}
          </p>
        </div>
      )}

      <Modal
        open={modalState?.type === "edit" || modalState?.type === "create"}
        onClose={close}
      >
        <Modal.Header onClose={close}>
          <span className="text-sm font-semibold text-gray-900 dark:text-gray-50">
            {modalState?.type === "edit"
              ? t("categories.table.editCategory")
              : modalState?.type === "create" &&
                  modalState.initialParentId !== null
                ? t("categories.table.createSubcategory")
                : t("categories.table.createCategory")}
          </span>
        </Modal.Header>
        <Modal.Body>
          <div className="space-y-4">
            {renderFormFields(formData, setFormData)}
            <div>
              <label
                htmlFor="parent-category"
                className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                {t("categories.table.parentCategory")}
              </label>
              <select
                id="parent-category"
                value={selectedParentId}
                onChange={(e) => setSelectedParentId(e.target.value)}
                className={selectClasses}
              >
                {getParentOptions(
                  modalState?.type === "edit" ? modalState.category.id : null,
                ).map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="outline" size="sm" onClick={close}>
            {t("common.cancel")}
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={
              modalState?.type === "edit" ? handleEditSave : handleCreateSave
            }
          >
            {t("common.save")}
          </Button>
        </Modal.Footer>
      </Modal>

      <Dialog
        open={modalState?.type === "delete"}
        onClose={close}
        title={t("categories.table.deleteCategory")}
        description={
          deleteTarget
            ? hasSubcategories
              ? t("categories.table.deleteDescriptionWithChildren", {
                  name: getCategoryLabel(deleteTarget),
                })
              : t("categories.table.deleteDescription", {
                  name: getCategoryLabel(deleteTarget),
                })
            : ""
        }
        onConfirm={handleDelete}
        confirmLabel={t("common.delete")}
        confirmVariant="destructive"
      />
    </div>
  );
};

export default CategoryTable;
