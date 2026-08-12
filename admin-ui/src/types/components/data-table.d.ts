type DataTableColumnType =
  | "string"
  | "number"
  | "money"
  | "date"
  | "boolean"
  | "chip"
  | "progress"
  | "actions";

interface DataTableActionButton<T> {
  icon: IconElement;
  label: string;
  danger?: boolean;
  variant?: "primary" | "success" | "warning" | "destructive";
  disabled?: boolean | ((row: T) => boolean);
  onClick: (row: T) => void;
}

interface DataTableMoneyOptions {
  currency?: "TRY" | "USD" | "EUR";
}

interface DataTableDateOptions {
  format?: "date" | "datetime" | "time";
}

interface DataTableChipOptions {
  colors: Record<string, string>;
}

interface DataTableNumberOptions {
  precision?: number;
  format?: "number" | "percent";
}

interface DataTableBooleanOptions {
  trueLabel?: string;
  falseLabel?: string;
}

interface DataTableProgressOptions {
  max?: number;
}

interface DataTableColumn<T> {
  id: string;
  header: string;
  accessor: ((row: T) => unknown) | string;
  type?: DataTableColumnType;
  sortable?: boolean;
  filterable?: boolean;
  visible?: boolean;
  width?: string | number;
  minWidth?: string | number;
  align?: "left" | "center" | "right";
  cell?: (value: unknown, row: T) => React.ReactNode;
  onClick?: (row: T) => void;
  moneyOptions?: DataTableMoneyOptions;
  dateOptions?: DataTableDateOptions;
  chipOptions?: DataTableChipOptions;
  numberOptions?: DataTableNumberOptions;
  booleanOptions?: DataTableBooleanOptions;
  progressOptions?: DataTableProgressOptions;
  labels?: Record<string, string>;
  actions?: DataTableActionButton<T>[];
}

interface DataTableContextMenuItem {
  id: string;
  label?: string;
  icon?: IconElement;
  disabled?: boolean;
  danger?: boolean;
  variant?: "primary" | "success" | "warning" | "destructive";
  separator?: boolean;
  onClick?: () => void;
}

interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  data: T[];
  completeData?: boolean;
  getRowId: (row: T) => string;
  selectable?: boolean;
  selectedIds?: string[];
  onSelectionChange?: (ids: string[]) => void;
  page?: number;
  pageSize?: number;
  totalCount?: number;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  pageSizeOptions?: number[];
  sortBy?: string;
  sortDir?: "asc" | "desc";
  onSortChange?: (columnId: string, dir: "asc" | "desc") => void;
  filters?: Record<string, unknown>;
  onFilterChange?: (columnId: string, value: unknown) => void;
  toolbarActions?: React.ReactNode;
  selectionActions?: (selectedRows: T[]) => React.ReactNode;
  renderRowContextMenu?: (row: T) => DataTableContextMenuItem[];
  onVisibleColumnsChange?: (columns: string[]) => void;
  isLoading?: boolean;
  isEmpty?: boolean;
  error?: string;
  className?: string;
  stickyHeader?: boolean;
  maxHeight?: string | number;
  hidePagination?: boolean;
}
