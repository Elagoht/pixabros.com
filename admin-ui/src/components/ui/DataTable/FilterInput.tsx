import { IconX } from "@tabler/icons-react";
import classNames from "classnames";
import type { ReactNode } from "react";
import { filterInputBase, filterSelectBase } from "@/utilities/constants";
import type {
  DateRangeValue,
  NumberFilter,
  NumberOp,
} from "@/utilities/filter";
import MultiselectDropdown from "./MultiselectDropdown";

export const filterInput = <T,>({
  colDef,
  filterValue,
  onFilterChange,
}: {
  colDef: DataTableColumn<T>;
  filterValue: unknown;
  onFilterChange: (value: unknown) => void;
}): ReactNode => {
  const type = colDef.type ?? "string";
  const hasFilter = filterValue !== undefined;

  const clearFilter = () => onFilterChange(undefined);

  const wrapperClasses = "flex items-center gap-1";

  switch (type) {
    case "string": {
      const val = String(filterValue ?? "");
      return (
        <div className={wrapperClasses}>
          <input
            type="text"
            value={val}
            onChange={(e) => onFilterChange(e.target.value || undefined)}
            placeholder="Filtrele..."
            className={filterInputBase}
          />
          {hasFilter && (
            <button
              type="button"
              onClick={clearFilter}
              className="shrink-0 rounded p-0.5 text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 transition-colors"
            >
              <IconX size={12} />
            </button>
          )}
        </div>
      );
    }

    case "number":
    case "money":
    case "progress": {
      const fv = (filterValue as NumberFilter) ?? {};
      const ops: NumberOp[] = ["=", ">", ">=", "<", "<="];
      return (
        <div className="flex gap-1">
          <select
            value={fv.op ?? ""}
            onChange={(e) =>
              onFilterChange({
                ...fv,
                op: (e.target.value || undefined) as NumberOp | undefined,
              })
            }
            className={classNames(
              filterSelectBase,
              "!w-12 shrink-0 text-center",
            )}
          >
            <option value="">--</option>
            {ops.map((op) => (
              <option key={op} value={op}>
                {op}
              </option>
            ))}
          </select>
          <input
            type="number"
            value={fv.value ?? ""}
            onChange={(e) =>
              onFilterChange({
                ...fv,
                value: e.target.value ? Number(e.target.value) : undefined,
              })
            }
            placeholder="Değer"
            className={classNames(filterInputBase, "min-w-[5rem] flex-1")}
          />
          {(fv.op !== undefined || fv.value !== undefined) && (
            <button
              type="button"
              onClick={() => onFilterChange(undefined)}
              className="shrink-0 rounded p-0.5 text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 transition-colors"
            >
              <IconX size={12} />
            </button>
          )}
        </div>
      );
    }

    case "date": {
      const range = (filterValue as DateRangeValue) ?? {};
      const hasDateFilter =
        range.start !== undefined || range.end !== undefined;
      return (
        <div className="flex gap-1">
          <input
            type="date"
            value={range.start ?? ""}
            onChange={(e) =>
              onFilterChange({
                ...range,
                start: e.target.value || undefined,
              })
            }
            className={classNames(filterInputBase, "w-1/2")}
          />
          <input
            type="date"
            value={range.end ?? ""}
            onChange={(e) =>
              onFilterChange({
                ...range,
                end: e.target.value || undefined,
              })
            }
            className={classNames(filterInputBase, "w-1/2")}
          />
          {hasDateFilter && (
            <button
              type="button"
              onClick={() => onFilterChange(undefined)}
              className="shrink-0 rounded p-0.5 text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 transition-colors"
            >
              <IconX size={12} />
            </button>
          )}
        </div>
      );
    }

    case "boolean": {
      const val = String(filterValue ?? "all");
      return (
        <div className={wrapperClasses}>
          <select
            value={val}
            onChange={(e) =>
              onFilterChange(
                e.target.value === "all" ? undefined : e.target.value,
              )
            }
            className={filterSelectBase}
          >
            <option value="all">All</option>
            <option value="true">
              {colDef.booleanOptions?.trueLabel ?? "True"}
            </option>
            <option value="false">
              {colDef.booleanOptions?.falseLabel ?? "False"}
            </option>
          </select>
          {val !== "all" && (
            <button
              type="button"
              onClick={() => onFilterChange(undefined)}
              className="shrink-0 rounded p-0.5 text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 transition-colors"
            >
              <IconX size={12} />
            </button>
          )}
        </div>
      );
    }

    case "chip": {
      const selected = (filterValue as string[]) ?? [];
      return (
        <div className={wrapperClasses}>
          <MultiselectDropdown
            options={Object.keys(colDef.chipOptions?.colors ?? {})}
            labels={colDef.labels}
            selected={selected}
            onChange={(values) =>
              onFilterChange(values.length > 0 ? values : undefined)
            }
          />
          {selected.length > 0 && (
            <button
              type="button"
              onClick={() => onFilterChange(undefined)}
              className="shrink-0 rounded p-0.5 text-gray-400 dark:text-gray-500 hover:text-red-500 dark:hover:text-red-400 transition-colors"
            >
              <IconX size={12} />
            </button>
          )}
        </div>
      );
    }

    default:
      return null;
  }
};
