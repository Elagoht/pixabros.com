export type NumberOp = "=" | ">" | ">=" | "<" | "<=";
export type NumberFilter = { op?: NumberOp; value?: number };
export type DateRangeValue = { start?: string; end?: string };

type FilterFnType = (
  row: { getValue: (id: string) => unknown },
  columnId: string,
  filterValue: unknown,
) => boolean;

export const numberFilterFn: FilterFnType = (row, columnId, filterValue) => {
  if (!filterValue || typeof filterValue !== "object") {
    return true;
  }
  const fv = filterValue as NumberFilter;
  if (fv.value === undefined || !fv.op) {
    return true;
  }
  const cellValue = Number(row.getValue(columnId));
  if (Number.isNaN(cellValue)) {
    return true;
  }
  switch (fv.op) {
    case "=":
      return cellValue === fv.value;
    case ">":
      return cellValue > fv.value;
    case ">=":
      return cellValue >= fv.value;
    case "<":
      return cellValue < fv.value;
    case "<=":
      return cellValue <= fv.value;
    default:
      return true;
  }
};

export const dateFilterFn: FilterFnType = (row, columnId, filterValue) => {
  if (!filterValue || typeof filterValue !== "object") {
    return true;
  }
  const fv = filterValue as DateRangeValue;
  const cellValue = new Date(row.getValue(columnId) as string).getTime();
  if (Number.isNaN(cellValue)) {
    return true;
  }
  if (fv.start && new Date(fv.start).getTime() > cellValue) {
    return false;
  }
  if (fv.end && new Date(fv.end).getTime() < cellValue) {
    return false;
  }
  return true;
};

export const booleanFilterFn: FilterFnType = (row, columnId, filterValue) => {
  if (!filterValue || filterValue === "all") {
    return true;
  }
  const cellValue = row.getValue(columnId);
  return filterValue === "true" ? cellValue === true : cellValue === false;
};

export const chipFilterFn: FilterFnType = (row, columnId, filterValue) => {
  if (
    !(filterValue && Array.isArray(filterValue)) ||
    filterValue.length === 0
  ) {
    return true;
  }
  const cellValue = String(row.getValue(columnId) ?? "");
  return filterValue.includes(cellValue);
};

export const filterFnMap: Partial<Record<DataTableColumnType, FilterFnType>> = {
  number: numberFilterFn,
  money: numberFilterFn,
  progress: numberFilterFn,
  date: dateFilterFn,
  boolean: booleanFilterFn,
  chip: chipFilterFn,
};
