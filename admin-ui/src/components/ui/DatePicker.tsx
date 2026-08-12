import {
  IconCalendar,
  IconChevronLeft,
  IconChevronRight,
  IconPointFilled,
} from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { type FC, useCallback, useState } from "react";
import { useI18n } from "@/lib/stores/i18n";

interface DatePickerProps {
  name: string;
  label?: string;
  className?: string;
}

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

const formatDateDisplay = (dateStr: string, locale: Locale): string => {
  const d = new Date(`${dateStr}T00:00:00`);
  return d.toLocaleDateString(locale === "tr" ? "tr-TR" : "en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
};

type View = "days" | "months" | "years";

const DatePicker: FC<DatePickerProps> = ({ name, label, className }) => {
  const [field, meta, helpers] = useField<string>(name);
  const hasError = meta.touched && !!meta.error;
  const { locale, t } = useI18n();
  const { days, months } = localeData[locale] ?? localeData.en;

  const [open, setOpen] = useState(false);
  const [view, setView] = useState<View>("days");

  const now = new Date();
  const selected = field.value ? new Date(`${field.value}T00:00:00`) : null;
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
    helpers.setValue(`${y}-${m}-${d}`);
    setOpen(false);
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
    <div className={classNames("w-full", className)}>
      {label && (
        <label
          htmlFor={name}
          className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300"
        >
          {label}
        </label>
      )}
      <div className="relative">
        <button
          type="button"
          id={name}
          onClick={() => setOpen((v) => !v)}
          className={classNames(
            "flex w-full items-center gap-2 rounded-lg border bg-gray-50 px-3 py-2 text-left text-sm transition-all duration-200",
            "text-gray-900 dark:bg-gray-800/50 dark:text-gray-50",
            "hover:ring-2 hover:ring-primary-400/30 hover:ring-offset-1",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
            hasError
              ? "border-red-500 focus-visible:ring-red-500 focus-visible:ring-offset-white dark:focus-visible:ring-offset-gray-950"
              : "border-gray-200 focus-visible:ring-primary-500 focus-visible:ring-offset-white dark:border-gray-700 dark:focus-visible:ring-primary-500 dark:focus-visible:ring-offset-gray-950",
            !field.value && "text-gray-400 dark:text-gray-500",
          )}
        >
          <IconCalendar
            size={16}
            className="shrink-0 text-gray-400 dark:text-gray-500"
          />
          <span className="flex-1 truncate">
            {field.value
              ? formatDateDisplay(field.value, locale)
              : t("common.selectDate")}
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
            <div
              className={classNames(
                "absolute z-50 mt-1 w-72 rounded-xl border border-gray-200/60 bg-white p-3 shadow-lg",
                "dark:border-gray-700/60 dark:bg-gray-900",
              )}
            >
              {/* Header */}
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

              {/* Today button */}
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

              {/* Days view */}
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

              {/* Months view */}
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

              {/* Years view */}
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
      {hasError && (
        <p className="mt-1.5 text-xs text-red-500 dark:text-red-400">
          {meta.error}
        </p>
      )}
    </div>
  );
};

export default DatePicker;
