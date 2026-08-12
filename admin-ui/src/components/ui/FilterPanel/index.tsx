import {
  IconCalendar,
  IconCheck,
  IconChevronLeft,
  IconChevronRight,
  IconFilter,
  IconPointFilled,
  IconX,
} from "@tabler/icons-react";
import classNames from "classnames";
import {
  type ChangeEvent,
  type FC,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { useSearchParams } from "react-router-dom";
import { useI18n } from "@/lib/stores/i18n";
import { checkboxBase } from "@/utilities/constants";
import type { FilterDef } from "./types";

const DAYS_EN = ["Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"];
const DAYS_TR = ["Pzr", "Pzt", "Sal", "Car", "Per", "Cum", "Cmt"];
const DAYS_RU = ["Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"];
const MONTHS_EN = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];
const MONTHS_TR = [
  "Ocak",
  "Şubat",
  "Mart",
  "Nisan",
  "Mayıs",
  "Haziran",
  "Temmuz",
  "Ağustos",
  "Eylül",
  "Ekim",
  "Kasım",
  "Aralık",
];
const MONTHS_RU = [
  "Январь",
  "Февраль",
  "Март",
  "Апрель",
  "Май",
  "Июнь",
  "Июль",
  "Август",
  "Сентябрь",
  "Октябрь",
  "Ноябрь",
  "Декабрь",
];

const localeData: Record<string, { days: string[]; months: string[] }> = {
  en: { days: DAYS_EN, months: MONTHS_EN },
  tr: { days: DAYS_TR, months: MONTHS_TR },
  ru: { days: DAYS_RU, months: MONTHS_RU },
};

const getDaysInMonth = (y: number, m: number) =>
  new Date(y, m + 1, 0).getDate();
const getFirstDay = (y: number, m: number) => new Date(y, m, 1).getDay();

const formatDateDisplay = (dateStr: string, locale: string): string => {
  const d = new Date(`${dateStr}T00:00:00`);
  return d.toLocaleDateString(locale === "tr" ? "tr-TR" : "en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
};

type CalendarView = "days" | "months" | "years";

interface FilterDatePickerProps {
  value: string;
  onChange: (value: string) => void;
}

const FilterDatePicker: FC<FilterDatePickerProps> = ({ value, onChange }) => {
  const { locale, t } = useI18n();
  const { days, months } = localeData[locale] ?? localeData.en;
  const [open, setOpen] = useState(false);
  const [view, setView] = useState<CalendarView>("days");

  const now = new Date();
  const selected = value ? new Date(`${value}T00:00:00`) : null;
  const [viewYear, setViewYear] = useState(
    selected?.getFullYear() ?? now.getFullYear(),
  );
  const [viewMonth, setViewMonth] = useState(
    selected?.getMonth() ?? now.getMonth(),
  );

  const yearPage = Math.floor(viewYear / 10) * 10;
  const yearsInPage = Array.from({ length: 12 }, (_, i) => yearPage - 1 + i);
  const daysInMonth = getDaysInMonth(viewYear, viewMonth);
  const firstDay = getFirstDay(viewYear, viewMonth);

  const prevMonth = useCallback(() => {
    if (viewMonth === 0) {
      setViewMonth(11);
      setViewYear((y) => y - 1);
    } else {
      setViewMonth((m) => m - 1);
    }
  }, [viewMonth]);

  const nextMonth = useCallback(() => {
    if (viewMonth === 11) {
      setViewMonth(0);
      setViewYear((y) => y + 1);
    } else {
      setViewMonth((m) => m + 1);
    }
  }, [viewMonth]);

  const prevYearPage = () => setViewYear((y) => y - 10);
  const nextYearPage = () => setViewYear((y) => y + 10);

  const selectDay = (day: number) => {
    const y = viewYear;
    const m = String(viewMonth + 1).padStart(2, "0");
    const d = String(day).padStart(2, "0");
    onChange(`${y}-${m}-${d}`);
    setOpen(false);
    setView("days");
  };

  const isToday = (day: number) =>
    viewYear === now.getFullYear() &&
    viewMonth === now.getMonth() &&
    day === now.getDate();
  const isSelected = (day: number) =>
    selected !== null &&
    viewYear === selected.getFullYear() &&
    viewMonth === selected.getMonth() &&
    day === selected.getDate();

  const cellCls = (active: boolean, extra?: boolean) =>
    classNames(
      "rounded-lg py-1.5 text-xs transition-all duration-200",
      active && "bg-primary-500 text-white font-medium",
      !active &&
        extra &&
        "border border-primary-400 text-primary-600 font-medium bg-primary-50 dark:bg-primary-900/15 dark:text-primary-400",
      !(active || extra) &&
        "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800",
    );

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={classNames(
          "flex w-full items-center gap-2 rounded-md border bg-white px-3 py-2 text-left text-sm shadow-inner transition duration-150 ease-out",
          "text-gray-900 dark:bg-gray-900 dark:text-gray-50",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950",
          !value && "text-gray-400 dark:text-gray-500",
          "border-gray-200 dark:border-gray-700",
        )}
      >
        <IconCalendar
          size={16}
          className="shrink-0 text-gray-400 dark:text-gray-500"
        />
        <span className="flex-1 truncate">
          {value ? formatDateDisplay(value, locale) : t("common.selectDate")}
        </span>
      </button>

      {open && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => {
              setOpen(false);
              setView("days");
            }}
          />
          <div className="absolute z-50 mt-1 w-72 rounded-xl border border-gray-200/60 bg-white p-3 shadow-lg dark:border-gray-700/60 dark:bg-gray-900">
            <div className="mb-2 flex items-center justify-between">
              <button
                type="button"
                onClick={
                  view === "years"
                    ? prevYearPage
                    : view === "months"
                      ? () => setViewYear((y) => y - 1)
                      : prevMonth
                }
                className="rounded p-1 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
              >
                <IconChevronLeft size={16} />
              </button>
              <button
                type="button"
                onClick={() => {
                  if (view === "days") {
                    setView("months");
                  } else if (view === "months") {
                    setView("years");
                  }
                }}
                className="text-sm font-medium text-gray-700 hover:text-primary-500 dark:text-gray-300 dark:hover:text-primary-500"
              >
                {view === "days" && `${months[viewMonth]} ${viewYear}`}
                {view === "months" && viewYear}
                {view === "years" && `${yearsInPage[1]} – ${yearsInPage[10]}`}
              </button>
              <button
                type="button"
                onClick={
                  view === "years"
                    ? nextYearPage
                    : view === "months"
                      ? () => setViewYear((y) => y + 1)
                      : nextMonth
                }
                className="rounded p-1 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
              >
                <IconChevronRight size={16} />
              </button>
            </div>

            {view === "days" && (
              <div className="mb-1">
                <button
                  type="button"
                  onClick={() => {
                    setViewYear(now.getFullYear());
                    setViewMonth(now.getMonth());
                  }}
                  className="inline-flex items-center gap-1 text-xs font-medium text-primary-500 hover:text-primary-500 dark:text-primary-300 dark:hover:text-primary-500"
                >
                  <IconPointFilled size={10} />
                  {t("common.today")}
                </button>
              </div>
            )}

            {view === "days" && (
              <div className="grid grid-cols-7 gap-0.5 text-center text-xs">
                {days.map((d) => (
                  <span
                    key={d}
                    className="py-1 font-medium text-gray-400 dark:text-gray-500"
                  >
                    {d}
                  </span>
                ))}
                {Array.from({ length: firstDay }).map((_, i) => (
                  <span key={`e-${i}`} />
                ))}
                {Array.from({ length: daysInMonth }).map((_, i) => {
                  const day = i + 1;
                  return (
                    <button
                      key={day}
                      type="button"
                      onClick={() => selectDay(day)}
                      className={cellCls(isSelected(day), isToday(day))}
                    >
                      {day}
                    </button>
                  );
                })}
              </div>
            )}

            {view === "months" && (
              <div className="grid grid-cols-3 gap-1 text-center text-xs">
                {months.map((m, i) => (
                  <button
                    key={i}
                    type="button"
                    onClick={() => {
                      setViewMonth(i);
                      setView("days");
                    }}
                    className={classNames(
                      "rounded-lg py-2 transition-all duration-200",
                      i === viewMonth &&
                        "bg-primary-500 font-medium text-white",
                      i !== viewMonth &&
                        "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800",
                    )}
                  >
                    {m}
                  </button>
                ))}
              </div>
            )}

            {view === "years" && (
              <div className="grid grid-cols-3 gap-1 text-center text-xs">
                {yearsInPage.map((y) => {
                  const offPage = y < yearPage || y >= yearPage + 10;
                  return (
                    <button
                      key={y}
                      type="button"
                      onClick={() => {
                        setViewYear(y);
                        setView("months");
                      }}
                      className={classNames(
                        "rounded-md py-1.5 transition duration-100",
                        y === viewYear &&
                          "bg-primary-500 font-medium text-white",
                        y !== viewYear &&
                          offPage &&
                          "text-gray-400 hover:bg-gray-100 dark:text-gray-600 dark:hover:bg-gray-800",
                        y !== viewYear &&
                          !offPage &&
                          "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800",
                      )}
                    >
                      {y}
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
};

interface FilterPanelProps {
  filters: FilterDef[];
  className?: string;
  title?: string;
  clearLabel?: string;
}

const inputClass =
  "w-full rounded-md border border-gray-200 bg-white px-3 py-2 text-sm shadow-inner transition duration-150 ease-out " +
  "text-gray-900 placeholder-gray-400 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-50 dark:placeholder-gray-500 " +
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950";

const sectionLabelClass =
  "mb-2.5 block text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400";

const DEBOUNCE_MS = 500;

const FilterPanel: FC<FilterPanelProps> = ({
  filters,
  className,
  title = "Filters",
  clearLabel = "Clear all",
}) => {
  const [searchParams, setSearchParams] = useSearchParams();

  const [localValues, setLocalValues] = useState<Record<string, string>>(() => {
    const vals: Record<string, string> = {};
    for (const filter of filters) {
      if (filter.type === "text" || filter.type === "date") {
        vals[filter.key] = searchParams.get(filter.key) ?? "";
      } else if (filter.type === "number-range") {
        vals[`${filter.key}_min`] = searchParams.get(`${filter.key}_min`) ?? "";
        vals[`${filter.key}_max`] = searchParams.get(`${filter.key}_max`) ?? "";
      }
    }
    return vals;
  });

  const timers = useRef<Record<string, ReturnType<typeof setTimeout>>>({});

  useEffect(() => {
    const currentTimers = timers.current;
    return () => {
      for (const timer of Object.values(currentTimers)) {
        clearTimeout(timer);
      }
    };
  }, []);

  const updateParam = (key: string, value: string) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (value) {
          next.set(key, value);
        } else {
          next.delete(key);
        }
        next.set("page", "1");
        return next;
      },
      { replace: true },
    );
  };

  const debouncedUpdate = (key: string, value: string) => {
    setLocalValues((prev) => ({ ...prev, [key]: value }));
    clearTimeout(timers.current[key]);
    timers.current[key] = setTimeout(
      () => updateParam(key, value),
      DEBOUNCE_MS,
    );
  };

  const activeCount = filters.reduce((count, filter) => {
    if (filter.type === "number-range") {
      return (
        count +
        (searchParams.has(`${filter.key}_min`) ||
        searchParams.has(`${filter.key}_max`)
          ? 1
          : 0)
      );
    }
    return count + (searchParams.has(filter.key) ? 1 : 0);
  }, 0);

  const clearAll = () => {
    const cleared: Record<string, string> = {};
    for (const key of Object.keys(localValues)) {
      cleared[key] = "";
    }
    setLocalValues(cleared);

    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        for (const filter of filters) {
          if (filter.type === "number-range") {
            next.delete(`${filter.key}_min`);
            next.delete(`${filter.key}_max`);
          } else {
            next.delete(filter.key);
          }
        }
        next.set("page", "1");
        return next;
      },
      { replace: true },
    );
  };

  const renderFilter = (filter: FilterDef) => {
    if (filter.type === "text") {
      return (
        <input
          type="text"
          placeholder={filter.placeholder}
          value={localValues[filter.key] ?? ""}
          onChange={(e: ChangeEvent<HTMLInputElement>) =>
            debouncedUpdate(filter.key, e.target.value)
          }
          className={inputClass}
        />
      );
    }

    if (filter.type === "date") {
      const value = searchParams.get(filter.key) ?? "";
      return (
        <FilterDatePicker
          value={value}
          onChange={(v) => updateParam(filter.key, v)}
        />
      );
    }

    if (filter.type === "select") {
      const value = searchParams.get(filter.key) ?? "";
      return (
        <select
          value={value}
          onChange={(e: ChangeEvent<HTMLSelectElement>) =>
            updateParam(filter.key, e.target.value)
          }
          disabled={filter.disabled}
          className={classNames(
            inputClass,
            filter.disabled && "opacity-30 cursor-not-allowed",
          )}
        >
          <option value="">{filter.placeholder ?? "All"}</option>
          {filter.options.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      );
    }

    if (filter.type === "multiselect") {
      const selected = (searchParams.get(filter.key) ?? "")
        .split(",")
        .filter(Boolean);

      const toggle = (optValue: string) => {
        const next = selected.includes(optValue)
          ? selected.filter((v) => v !== optValue)
          : [...selected, optValue];
        updateParam(filter.key, next.join(","));
      };

      return (
        <div className="space-y-2">
          {filter.options.map((opt) => {
            const checked = selected.includes(opt.value);
            return (
              <label
                key={opt.value}
                className="flex cursor-pointer items-center gap-2.5"
              >
                <div className="relative flex shrink-0 items-center justify-center">
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => toggle(opt.value)}
                    className={classNames(
                      checkboxBase,
                      checked &&
                        "border-primary-500 bg-primary-500 dark:bg-primary-500",
                    )}
                  />
                  {checked && (
                    <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
                      <IconCheck
                        size={10}
                        strokeWidth={3}
                        className="text-white"
                      />
                    </div>
                  )}
                </div>
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  {opt.label}
                </span>
              </label>
            );
          })}
        </div>
      );
    }

    if (filter.type === "number-range") {
      const minKey = `${filter.key}_min`;
      const maxKey = `${filter.key}_max`;
      return (
        <div className="flex items-center gap-2">
          <input
            type="number"
            placeholder={filter.min === undefined ? "Min" : String(filter.min)}
            min={filter.min}
            max={filter.max}
            step={filter.step ?? 1}
            value={localValues[minKey] ?? ""}
            onChange={(e: ChangeEvent<HTMLInputElement>) =>
              debouncedUpdate(minKey, e.target.value)
            }
            className={inputClass}
          />
          <span className="shrink-0 text-xs text-gray-400">–</span>
          <input
            type="number"
            placeholder={filter.max === undefined ? "Max" : String(filter.max)}
            min={filter.min}
            max={filter.max}
            step={filter.step ?? 1}
            value={localValues[maxKey] ?? ""}
            onChange={(e: ChangeEvent<HTMLInputElement>) =>
              debouncedUpdate(maxKey, e.target.value)
            }
            className={inputClass}
          />
        </div>
      );
    }

    return null;
  };

  return (
    <div
      className={classNames(
        "rounded-lg border self-start md:sticky md:top-6 max-h-[calc(100dvh-7rem)] overflow-y-auto border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900",
        className,
      )}
    >
      <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <IconFilter size={15} className="text-gray-500 dark:text-gray-400" />
          <span className="text-sm font-medium text-gray-900 dark:text-gray-50">
            {title}
          </span>
          {activeCount > 0 && (
            <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-primary-500 px-1.5 text-xs font-medium text-white">
              {activeCount}
            </span>
          )}
        </div>
        {activeCount > 0 && (
          <button
            type="button"
            onClick={clearAll}
            className="flex items-center gap-1 text-xs text-gray-500 transition hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <IconX size={12} />
            {clearLabel}
          </button>
        )}
      </div>

      <div className="divide-y divide-gray-100 dark:divide-gray-800">
        {filters.map((filter) => (
          <div key={filter.key} className="px-4 py-4">
            <span className={sectionLabelClass}>{filter.label}</span>
            {renderFilter(filter)}
          </div>
        ))}
      </div>
    </div>
  );
};

export default FilterPanel;
