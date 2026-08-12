import classNames from "classnames";
import type { ReactNode } from "react";
import { Button, Tooltip } from "@/components/ui";
import { chipColors } from "@/utilities/constants";
import {
  formatDate,
  formatMoney,
  formatNumber,
} from "@/utilities/localization";

export {
  formatDate,
  formatMoney,
  formatNumber,
} from "@/utilities/localization";

export const renderChip = (
  value: unknown,
  options?: DataTableChipOptions,
  labels?: Record<string, string>,
): ReactNode => {
  const str = String(value ?? "");
  const label = labels?.[str] ?? str;
  const colorKey = options?.colors?.[str] ?? "gray";
  const colorClass = chipColors[colorKey] ?? chipColors.gray;

  return (
    <span
      className={classNames(
        "inline-block animate-pulse rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset",
        colorClass,
      )}
      style={{ animationDuration: "3s" }}
    >
      {label}
    </span>
  );
};

export const renderBoolean = (
  value: unknown,
  options?: DataTableBooleanOptions,
): ReactNode => {
  const isTrue =
    value === true || value === "true" || value === 1 || value === "1";
  if (isTrue) {
    return (
      <span className="inline-flex items-center gap-1 text-xs font-medium text-green-600 dark:text-green-400">
        <span className="h-1.5 w-1.5 rounded-full bg-green-500" />
        {options?.trueLabel ?? "True"}
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 text-xs font-medium text-red-600 dark:text-red-400">
      <span className="h-1.5 w-1.5 rounded-full bg-red-500" />
      {options?.falseLabel ?? "False"}
    </span>
  );
};

export const renderProgress = (
  value: unknown,
  options?: DataTableProgressOptions,
): ReactNode => {
  const num = Number(value);
  if (Number.isNaN(num)) {
    return String(value ?? "");
  }
  const clamped = Math.min(100, Math.max(0, num));
  const max = options?.max ?? 100;
  const pct = Math.round((clamped / max) * 100);

  return (
    <div className="flex items-center gap-2">
      <div className="h-2 w-full max-w-[6rem] overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
        <div
          className="h-full rounded-full bg-primary-500 transition-all duration-300"
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-xs tabular-nums text-gray-500 dark:text-gray-400">
        {pct}%
      </span>
    </div>
  );
};

export const safeFormatDate = (value: unknown): string => {
  if (!value) {
    return "-";
  }
  if (typeof value === "string") {
    const d = new Date(value);
    return Number.isNaN(d.getTime()) ? "-" : d.toLocaleDateString("tr-TR");
  }
  if (value instanceof Date) {
    return Number.isNaN(value.getTime())
      ? "-"
      : value.toLocaleDateString("tr-TR");
  }
  return "-";
};

export const renderActions = <T,>(
  row: T,
  actions?: DataTableActionButton<T>[],
): ReactNode => {
  if (!actions || actions.length === 0) {
    return null;
  }

  return (
    <div className="flex items-center justify-end gap-0.5">
      {actions.map((action, i) => {
        const Icon = action.icon;
        const isDisabled =
          typeof action.disabled === "function"
            ? action.disabled(row)
            : action.disabled;
        const button = (
          <Button
            variant={action.variant ?? "ghost"}
            size="sm"
            disabled={isDisabled}
            onClick={(e) => {
              e.stopPropagation();
              action.onClick(row);
            }}
            className={classNames(
              "!rounded !p-1.5",
              isDisabled && "pointer-events-none opacity-40",
            )}
          >
            <Icon size={16} />
          </Button>
        );

        // A disabled Button stops firing pointer events, which would also
        // swallow the tooltip's hover -- exactly when the label explaining
        // why it is unavailable matters most. The wrapper listens instead.
        return (
          <Tooltip key={i} content={action.label} position="top">
            {isDisabled ? (
              <span className="inline-flex">{button}</span>
            ) : (
              button
            )}
          </Tooltip>
        );
      })}
    </div>
  );
};

export const renderCellContent = <T,>(
  value: unknown,
  row: T,
  colDef: DataTableColumn<T>,
): ReactNode => {
  if (colDef.cell) {
    return colDef.cell(value, row);
  }

  switch (colDef.type) {
    case "money":
      return formatMoney(value, colDef.moneyOptions);
    case "date":
      return formatDate(value, colDef.dateOptions);
    case "boolean":
      return renderBoolean(value, colDef.booleanOptions);
    case "chip":
      return renderChip(value, colDef.chipOptions, colDef.labels);
    case "number":
      return formatNumber(value, colDef.numberOptions);
    case "progress":
      return renderProgress(value, colDef.progressOptions);
    case "actions":
      return renderActions(row, colDef.actions);
    default:
      return String(value ?? "");
  }
};
