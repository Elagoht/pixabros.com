import {
  IconAdjustments,
  IconArrowsSort,
  IconCheck,
  IconChevronDown,
  IconChevronUp,
  IconX,
} from "@tabler/icons-react";
import {
  type ColumnDef,
  type ColumnFiltersState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type PaginationState,
  type Row,
  type RowSelectionState,
  type SortingState,
  useReactTable,
  type VisibilityState,
} from "@tanstack/react-table";
import classNames from "classnames";
import {
  type CSSProperties,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { checkboxBase } from "@/utilities/constants";
import { Environment } from "@/utilities/environment";
import { filterFnMap } from "@/utilities/filter";
import { renderCellContent } from "@/utilities/format";
import Alert from "../Alert";
import EmptyState from "../EmptyState";
import Pagination from "../Pagination";
import Skeleton from "../Skeleton";
import { filterInput } from "./FilterInput";
import TableCheckbox from "./TableCheckbox";

const DataTable = <T,>({
  columns,
  data,
  completeData,
  getRowId,
  selectable,
  selectedIds,
  onSelectionChange,
  page: externalPage,
  pageSize: externalPageSize,
  totalCount,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions,
  sortBy: externalSortBy,
  sortDir: externalSortDir,
  onSortChange,
  toolbarActions,
  selectionActions,
  renderRowContextMenu,
  onVisibleColumnsChange,
  isLoading,
  isEmpty,
  error,
  className,
  stickyHeader,
  maxHeight,
  hidePagination,
}: DataTableProps<T>) => {
  const isClient = completeData === true;
  const sourceData = useMemo(() => data ?? [], [data]);
  const size = externalPageSize ?? Environment.pageSize;

  /* ── State ── */

  const [sorting, setSorting] = useState<SortingState>(() =>
    externalSortBy && externalSortDir
      ? [{ id: externalSortBy, desc: externalSortDir === "desc" }]
      : [],
  );

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: isClient ? 0 : externalPage ? externalPage - 1 : 0,
    pageSize: size,
  });

  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    row: T;
  } | null>(null);

  /* ── Sync external sorting → internal ── */

  useEffect(() => {
    if (!isClient && externalSortBy) {
      setSorting([{ id: externalSortBy, desc: externalSortDir === "desc" }]);
    }
  }, [isClient, externalSortBy, externalSortDir]);

  /* ── Sync external pagination → internal ── */

  useEffect(() => {
    if (!isClient && externalPage !== undefined) {
      setPagination((prev) => ({
        ...prev,
        pageIndex: externalPage - 1,
        pageSize: externalPageSize ?? prev.pageSize,
      }));
    }
  }, [isClient, externalPage, externalPageSize]);

  /* ── Sync external selectedIds → internal ── */

  useEffect(() => {
    if (selectedIds) {
      const sel: RowSelectionState = {};
      for (const row of sourceData) {
        const id = getRowId(row);
        if (selectedIds.includes(id)) {
          sel[id] = true;
        }
      }
      setRowSelection(sel);
    }
  }, [selectedIds, sourceData, getRowId]);

  /* ── Notify parent of selection changes ── */

  const prevSelectionRef = useRef<string[]>([]);
  useEffect(() => {
    if (onSelectionChange) {
      const ids = Object.entries(rowSelection)
        .filter(([, v]) => v)
        .map(([k]) => k);
      if (JSON.stringify(ids) !== JSON.stringify(prevSelectionRef.current)) {
        onSelectionChange(ids);
      }
      prevSelectionRef.current = ids;
    }
  }, [rowSelection, onSelectionChange]);

  /* ── Notify parent of visibility changes ── */

  useEffect(() => {
    onVisibleColumnsChange?.(
      columns
        .filter((col: DataTableColumn<T>) => columnVisibility[col.id] !== false)
        .map((col: DataTableColumn<T>) => col.id),
    );
  }, [columnVisibility, onVisibleColumnsChange, columns]);

  /* ── Column Definitions ── */

  const tanstackColDefs: ColumnDef<T>[] = useMemo(() => {
    const defs: ColumnDef<T>[] = [];

    if (selectable) {
      defs.push({
        id: "_select",
        header: ({ table }) => (
          <TableCheckbox
            checked={table.getIsAllRowsSelected()}
            indeterminate={table.getIsSomeRowsSelected()}
            onChange={table.getToggleAllRowsSelectedHandler()}
          />
        ),
        cell: ({ row }) => (
          <TableCheckbox
            checked={row.getIsSelected()}
            onChange={row.getToggleSelectedHandler()}
          />
        ),
        enableSorting: false,
        enableColumnFilter: false,
        enableHiding: false,
        size: 48,
        minSize: 48,
      });
    }

    for (const col of columns as DataTableColumn<T>[]) {
      const filterFn = col.type ? filterFnMap[col.type] : undefined;

      defs.push({
        id: col.id,
        accessorKey:
          typeof col.accessor === "string" ? col.accessor : undefined,
        accessorFn:
          typeof col.accessor === "function" ? col.accessor : undefined,
        header: col.header,
        enableSorting: col.sortable ?? true,
        enableHiding: col.type !== "actions",
        ...(filterFn && { filterFn }),
        size: typeof col.width === "number" ? col.width : undefined,
        minSize: typeof col.minWidth === "number" ? col.minWidth : undefined,
        cell: (info) => {
          const value = info.getValue();
          const row = info.row.original as T;
          if (col.onClick) {
            return (
              <button
                type="button"
                onClick={() => col.onClick?.(row)}
                className="w-full text-left outline-none hover:underline"
              >
                {renderCellContent(value, row, col)}
              </button>
            );
          }
          return renderCellContent(value, row, col);
        },
        meta: { colDef: col },
      });
    }

    return defs;
  }, [columns, selectable]);

  /* ── Table Instance ── */

  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable<T>({
    data: sourceData,
    columns: tanstackColDefs,
    getRowId: (row) => getRowId(row),
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
      pagination,
    },
    onSortingChange: (updater) => {
      const next = typeof updater === "function" ? updater(sorting) : updater;
      setSorting(next);

      if (!isClient) {
        const s = next[0];
        if (s && onSortChange) {
          onSortChange(s.id, s.desc ? "desc" : "asc");
        }
      }
    },
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    onPaginationChange: (updater) => {
      const next =
        typeof updater === "function" ? updater(pagination) : updater;
      const pageIndexChanged = next.pageIndex !== pagination.pageIndex;
      setPagination(next);

      if (!isClient && onPageChange && pageIndexChanged) {
        onPageChange(next.pageIndex + 1);
      }
    },
    enableRowSelection: selectable,
    enableMultiRowSelection: selectable,
    getCoreRowModel: getCoreRowModel(),
    ...(isClient &&
      !hidePagination && {
        getPaginationRowModel: getPaginationRowModel(),
      }),
    ...(isClient && {
      getSortedRowModel: getSortedRowModel(),
      getFilteredRowModel: getFilteredRowModel(),
    }),
    manualSorting: !isClient,
    manualPagination: !isClient || !!hidePagination,
    ...(!isClient && {
      pageCount: totalCount ? Math.ceil(totalCount / size) : undefined,
    }),
  });

  /* ── Render Helpers ── */

  const visibleColumns = table.getAllLeafColumns();
  const hasHideableColumns = visibleColumns.some((col) => col.getCanHide());

  /* ── Context Menu ── */

  const handleRowContextMenu = useCallback(
    (e: React.MouseEvent, row: Row<T>) => {
      if (!renderRowContextMenu) {
        return;
      }
      e.preventDefault();
      setContextMenu({
        x: e.clientX,
        y: e.clientY,
        row: row.original,
      });
    },
    [renderRowContextMenu],
  );

  useEffect(() => {
    if (!contextMenu) {
      return;
    }
    const close = () => setContextMenu(null);
    document.addEventListener("click", close);
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape") {
        close();
      }
    });
    return () => {
      document.removeEventListener("click", close);
      document.removeEventListener("keydown", close);
    };
  }, [contextMenu]);

  /* ── Column Visibility Toggle ── */

  const [showVisibilityMenu, setShowVisibilityMenu] = useState(false);
  const visibilityRef = useRef<HTMLDivElement>(null);
  const visibilityButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!showVisibilityMenu) {
      return;
    }
    const handler = (e: MouseEvent) => {
      if (
        visibilityRef.current &&
        !visibilityRef.current.contains(e.target as Node) &&
        visibilityButtonRef.current &&
        !visibilityButtonRef.current.contains(e.target as Node)
      ) {
        setShowVisibilityMenu(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showVisibilityMenu]);

  /* ── Empty & Error ── */

  if (error) {
    return (
      <Alert
        variant="error"
        title="Bir hata oluştu"
        description={error}
        className={className}
      />
    );
  }

  const rows = table.getRowModel().rows;
  const columnCount = table.getAllLeafColumns().length;
  const showEmpty = !isLoading && (isEmpty ?? rows.length === 0);
  const showSkeleton = isLoading;

  /* ── Render ── */

  return (
    <div className={classNames("space-y-4", className)}>
      {/* Toolbar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {toolbarActions}

          {selectable && Object.keys(rowSelection).length > 0 && (
            <>
              <span className="text-sm text-gray-500 dark:text-gray-400">
                {
                  Object.keys(rowSelection).filter((k) => rowSelection[k])
                    .length
                }{" "}
                seçili
              </span>
              {selectionActions?.(
                table.getSelectedRowModel().rows.map((r) => r.original),
              )}
            </>
          )}
        </div>

        <div className="flex items-center gap-2">
          {columnFilters.length > 0 && (
            <>
              <span className="inline-flex items-center justify-center rounded-full bg-primary-500 px-2.5 py-0.5 text-xs font-bold text-white">
                {columnFilters.length}
              </span>
              <button
                type="button"
                onClick={() => setColumnFilters([])}
                className="flex items-center gap-1.5 rounded-lg border border-red-200/60 bg-red-50 px-3 py-2 text-xs font-medium text-red-600 transition-all duration-200 hover:ring-2 hover:ring-red-400/30 hover:ring-offset-1 dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400 dark:hover:bg-red-900/30"
              >
                <IconX size={14} />
                Filtreleri Temizle
              </button>
            </>
          )}

          {hasHideableColumns && (
            <div className="relative">
              <button
                ref={visibilityButtonRef}
                type="button"
                onClick={() => setShowVisibilityMenu((v) => !v)}
                className="flex items-center gap-1.5 rounded-lg border border-gray-200/60 bg-white px-3 py-2 text-xs font-medium text-gray-700 transition-all duration-200 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1 dark:border-gray-700/60 dark:bg-gray-900 dark:text-gray-300 dark:hover:bg-gray-800"
              >
                <IconAdjustments size={14} />
                Kolonlar
              </button>

              {showVisibilityMenu &&
                createPortal(
                  <div
                    ref={visibilityRef}
                    className="fixed z-50 min-w-[10rem] rounded-xl border border-gray-200/60 bg-white py-1 shadow-lg dark:border-gray-700/60 dark:bg-gray-900"
                    style={{
                      top: visibilityButtonRef.current
                        ? visibilityButtonRef.current.getBoundingClientRect()
                            .bottom + 4
                        : 0,
                      left: visibilityButtonRef.current
                        ? visibilityButtonRef.current.getBoundingClientRect()
                            .left
                        : 0,
                    }}
                  >
                    {columns
                      .filter(
                        (col: DataTableColumn<T>) => col.type !== "actions",
                      )
                      .map((col: DataTableColumn<T>) => {
                        const colId = col.id;
                        const isVisible = columnVisibility[colId] !== false;
                        return (
                          <button
                            key={colId}
                            type="button"
                            onClick={() => {
                              table
                                .getColumn(colId)
                                ?.toggleVisibility(!isVisible);
                            }}
                            className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800"
                          >
                            <span className="relative inline-flex items-center justify-center">
                              <input
                                type="checkbox"
                                checked={isVisible}
                                readOnly
                                className={classNames(
                                  checkboxBase,
                                  "h-3.5 w-3.5",
                                  isVisible &&
                                    "border-primary-500 bg-primary-500 dark:bg-primary-500",
                                )}
                              />
                              {isVisible && (
                                <IconCheck
                                  size={10}
                                  strokeWidth={3}
                                  className="pointer-events-none absolute text-white"
                                />
                              )}
                            </span>
                            <span className="flex-1">{col.header}</span>
                          </button>
                        );
                      })}
                  </div>,
                  document.body,
                )}
            </div>
          )}
        </div>
      </div>

      {/* Table Container */}
      <div
        className={classNames(
          "overflow-x-auto rounded-xl border border-gray-200/60 bg-white dark:border-gray-700/60 dark:bg-gray-900",
          maxHeight && "overflow-y-auto",
        )}
        style={maxHeight ? { maxHeight } : undefined}
      >
        <table className="w-full border-collapse text-sm">
          {/* Header */}
          <thead>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  const colMeta = header.column.columnDef.meta as
                    | { colDef: DataTableColumn<T> }
                    | undefined;
                  const colDef = colMeta?.colDef;
                  const canSort = header.column.getCanSort();
                  const sortState = header.column.getIsSorted();
                  const isSorted = sortState !== false;
                  const sortDir = isSorted
                    ? sortState === "asc"
                      ? "asc"
                      : "desc"
                    : false;

                  const colStyle: CSSProperties = {};
                  if (colDef?.width) {
                    colStyle.width =
                      typeof colDef.width === "number"
                        ? `${colDef.width}px`
                        : colDef.width;
                  }
                  if (colDef?.minWidth) {
                    colStyle.minWidth =
                      typeof colDef.minWidth === "number"
                        ? `${colDef.minWidth}px`
                        : colDef.minWidth;
                  }

                  return (
                    <th
                      key={header.id}
                      style={colStyle}
                      className={classNames(
                        "border-b border-gray-200/60 bg-gray-100 px-4 py-3 text-xs font-bold uppercase tracking-wider text-gray-600 dark:border-gray-700/60 dark:bg-gray-800 dark:text-gray-400",
                        canSort &&
                          "cursor-pointer select-none transition-colors hover:bg-gray-100 dark:hover:bg-gray-800",
                        colDef?.align === "center" && "text-center",
                        colDef?.align === "right" && "text-right",
                        stickyHeader && "sticky top-0 z-10",
                      )}
                      onClick={
                        canSort
                          ? header.column.getToggleSortingHandler()
                          : undefined
                      }
                    >
                      <div className="flex items-center gap-2">
                        <span className="flex-1">
                          {header.isPlaceholder
                            ? null
                            : flexRender(
                                header.column.columnDef.header,
                                header.getContext(),
                              )}
                        </span>
                        {canSort && (
                          <span className="inline-flex flex-col items-center leading-none text-gray-300 dark:text-gray-600">
                            {isSorted ? (
                              sortDir === "asc" ? (
                                <IconChevronUp
                                  size={14}
                                  className="text-primary-500"
                                />
                              ) : (
                                <IconChevronDown
                                  size={14}
                                  className="text-primary-500"
                                />
                              )
                            ) : (
                              <IconArrowsSort
                                size={12}
                                className="opacity-50 group-hover:opacity-100 transition-opacity"
                              />
                            )}
                          </span>
                        )}
                      </div>

                      {/* Inline Filter (client-side only) */}
                      {isClient &&
                        colDef &&
                        colDef.filterable !== false &&
                        header.column.getCanFilter() && (
                          <div
                            className="mt-1.5"
                            onClick={(e) => e.stopPropagation()}
                          >
                            {filterInput({
                              colDef,
                              filterValue: header.column.getFilterValue(),
                              onFilterChange: (val) =>
                                header.column.setFilterValue(val),
                            })}
                          </div>
                        )}
                    </th>
                  );
                })}
              </tr>
            ))}
          </thead>

          {/* Body */}
          <tbody>
            {showSkeleton &&
              Array.from({ length: Math.min(size, 5) }).map((_, i) => (
                <tr key={`skeleton-${i}`}>
                  {Array.from({ length: columnCount }).map((_, j) => (
                    <td
                      key={j}
                      className="border-b border-gray-100 px-3 py-2.5 dark:border-gray-700"
                    >
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))}

            {showEmpty && !showSkeleton && (
              <tr>
                <td colSpan={columnCount} className="px-3 py-12">
                  <EmptyState
                    title="Kayıt bulunamadı"
                    description="Henüz hiçbir veri eklenmemiş."
                  />
                </td>
              </tr>
            )}

            {!(showSkeleton || showEmpty) &&
              rows.map((row, rowIdx) => {
                const rowId = getRowId(row.original);
                return (
                  <tr
                    key={rowId}
                    onContextMenu={(e) => handleRowContextMenu(e, row)}
                    className={classNames(
                      "transition-all duration-300",
                      row.getIsSelected()
                        ? "bg-primary-50 dark:bg-primary-900/20"
                        : "bg-white hover:bg-gray-50 dark:bg-gray-950 dark:hover:bg-primary-900/8",
                    )}
                  >
                    {row.getVisibleCells().map((cell) => {
                      const colMeta = cell.column.columnDef.meta as
                        | { colDef: DataTableColumn<T> }
                        | undefined;
                      const colDef = colMeta?.colDef;

                      const colStyle: CSSProperties = {};
                      if (colDef?.width) {
                        colStyle.width =
                          typeof colDef.width === "number"
                            ? `${colDef.width}px`
                            : colDef.width;
                      }

                      return (
                        <td
                          key={cell.id}
                          style={colStyle}
                          className={classNames(
                            "whitespace-nowrap px-3 py-2.5 text-sm text-gray-700 dark:text-gray-300",
                            colDef?.align === "center" && "text-center",
                            colDef?.align === "right" && "text-right",
                            rowIdx < rows.length - 1 &&
                              "border-b border-gray-100 dark:border-gray-700",
                          )}
                        >
                          {flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext(),
                          )}
                        </td>
                      );
                    })}
                  </tr>
                );
              })}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {!(hidePagination || isLoading || showEmpty) &&
        (() => {
          const sizeOptions = pageSizeOptions ?? [10, 25, 50, 100];
          const currentSize = isClient
            ? table.getState().pagination.pageSize
            : size;

          const infoText = (() => {
            if (isClient) {
              const { pageIndex, pageSize: ps } = table.getState().pagination;
              const totalRows = table.getFilteredRowModel().rows.length;
              const start = totalRows === 0 ? 0 : pageIndex * ps + 1;
              const end = Math.min((pageIndex + 1) * ps, totalRows);
              return `${totalRows} içerikten ${start}–${end} arası gösteriliyor`;
            }
            if (totalCount === undefined) {
              return null;
            }
            const p = externalPage ?? 1;
            const start = Math.min((p - 1) * size + 1, totalCount);
            const end = Math.min(p * size, totalCount);
            return `${totalCount} içerikten ${start}–${end} arası gösteriliyor`;
          })();

          const handleSizeChange = (newSize: number) => {
            if (isClient) {
              table.setPageSize(newSize);
            } else {
              onPageSizeChange?.(newSize);
            }
          };

          return (
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between lg:gap-4">
              <span className="text-sm text-gray-500 dark:text-gray-400">
                {infoText}
              </span>

              <div className="flex items-center justify-between gap-3 lg:justify-end">
                <div className="flex items-center gap-1.5">
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    Sayfa başı
                  </span>
                  <select
                    value={currentSize}
                    onChange={(e) => handleSizeChange(Number(e.target.value))}
                    className="rounded-lg border border-gray-200/60 bg-white px-2 py-1.5 text-xs font-medium text-gray-700 transition-all duration-200 hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1 focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-gray-700/60 dark:bg-gray-900 dark:text-gray-300"
                  >
                    {sizeOptions.map((opt) => (
                      <option key={opt} value={opt}>
                        {opt}
                      </option>
                    ))}
                  </select>
                </div>

                {isClient ? (
                  <Pagination
                    page={table.getState().pagination.pageIndex + 1}
                    totalPages={table.getPageCount()}
                    onChange={(p) => table.setPageIndex(p - 1)}
                  />
                ) : (
                  totalCount !== undefined && (
                    <Pagination
                      page={externalPage ?? 1}
                      totalPages={Math.ceil(totalCount / size)}
                      onChange={(p) => onPageChange?.(p)}
                    />
                  )
                )}
              </div>
            </div>
          );
        })()}

      {/* Context Menu Portal */}
      {contextMenu &&
        createPortal(
          <div
            className="fixed z-50 min-w-[12rem] rounded-xl border border-gray-200/60 bg-white py-1 shadow-lg dark:border-gray-700/60 dark:bg-gray-900"
            style={{
              left: Math.min(contextMenu.x, window.innerWidth - 200),
              top: Math.min(contextMenu.y, window.innerHeight - 200),
            }}
          >
            {renderRowContextMenu?.(contextMenu.row)?.map((item) => {
              if (item.separator) {
                return (
                  <div
                    key={item.id}
                    className="my-1 border-t border-gray-200 dark:border-gray-700"
                  />
                );
              }

              const Icon = item.icon;

              return (
                <button
                  key={item.id}
                  type="button"
                  disabled={item.disabled}
                  onClick={() => {
                    if (item.disabled) {
                      return;
                    }
                    item.onClick?.();
                  }}
                  className={classNames(
                    "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm outline-none transition duration-100",
                    item.disabled && "pointer-events-none opacity-40",
                    !item.disabled &&
                      (() => {
                        const v =
                          item.variant ??
                          (item.danger ? "destructive" : undefined);
                        switch (v) {
                          case "primary":
                            return "text-primary-600 hover:bg-primary-50 dark:text-primary-400 dark:hover:bg-primary-900/10";
                          case "success":
                            return "text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-900/10";
                          case "warning":
                            return "text-amber-600 hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/10";
                          case "destructive":
                            return "text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/10";
                          default:
                            return "text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800";
                        }
                      })(),
                  )}
                >
                  {Icon && <Icon size={16} className="shrink-0" />}
                  {item.label}
                </button>
              );
            })}
          </div>,
          document.body,
        )}
    </div>
  );
};

export default DataTable;
