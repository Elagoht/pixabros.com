import {
  IconCalendar,
  IconChevronLeft,
  IconChevronRight,
  IconPointFilled,
} from "@tabler/icons-react";
import classNames from "classnames";
import { useField } from "formik";
import { type FC, useCallback, useEffect, useRef, useState } from "react";
import { useI18n } from "@/lib/stores/i18n";

interface DateTimePickerProps {
  name: string;
  label?: string;
  hour12?: boolean;
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

interface DTValue {
  date: string;
  hour: number;
  minute: number;
}

const parseValue = (raw: string): DTValue | null => {
  if (!raw) {
    return null;
  }
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) {
    return null;
  }
  return {
    date: `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`,
    hour: d.getHours(),
    minute: d.getMinutes(),
  };
};

const serialize = (v: DTValue): string =>
  `${v.date}T${String(v.hour).padStart(2, "0")}:${String(v.minute).padStart(2, "0")}:00`;

const formatDisplay = (v: DTValue, locale: Locale, hour12: boolean): string => {
  const [y, mo, d] = v.date.split("-").map(Number);
  const dt = new Date(y, mo - 1, d, v.hour, v.minute);
  const datePart = dt.toLocaleDateString(locale === "tr" ? "tr-TR" : "en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
  const timePart = dt.toLocaleTimeString(locale === "tr" ? "tr-TR" : "en-US", {
    hour: "2-digit",
    minute: "2-digit",
    hour12,
  });
  return `${datePart}, ${timePart}`;
};

type View = "days" | "months" | "years";
type TimeWheel = null | "hour" | "minute";

const DateTimePicker: FC<DateTimePickerProps> = ({
  name,
  label,
  hour12 = false,
  className,
}) => {
  const [field, meta, helpers] = useField<string>(name);
  const hasError = meta.touched && !!meta.error;
  const { locale, t } = useI18n();
  const { days, months } = localeData[locale] ?? localeData.en;

  const [open, setOpen] = useState(false);
  const [view, setView] = useState<View>("days");
  const parsed = parseValue(field.value);
  const now = new Date();

  const [viewYear, setViewYear] = useState(
    parsed
      ? new Date(`${parsed.date}T00:00:00`).getFullYear()
      : now.getFullYear(),
  );
  const [viewMonth, setViewMonth] = useState(
    parsed ? new Date(`${parsed.date}T00:00:00`).getMonth() : now.getMonth(),
  );
  const [hour, setHour] = useState(parsed?.hour ?? 9);
  const [minute, setMinute] = useState(parsed?.minute ?? 0);
  const [timeWheel, setTimeWheel] = useState<TimeWheel>(null);
  const wheelRef = useRef<HTMLDivElement>(null);

  const yearPage = Math.floor(viewYear / 10) * 10;
  const yearsInPage = Array.from({ length: 12 }, (_, i) => yearPage - 1 + i);
  const daysInMonth = getDaysInMonth(viewYear, viewMonth);
  const firstDay = getFirstDay(viewYear, viewMonth);

  useEffect(() => {
    if (!timeWheel) {
      return;
    }
    const handler = (e: MouseEvent) => {
      if (wheelRef.current && !wheelRef.current.contains(e.target as Node)) {
        setTimeWheel(null);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [timeWheel]);

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

  const selectDay = (day: number) => {
    const y = viewYear;
    const m = String(viewMonth + 1).padStart(2, "0");
    const d = String(day).padStart(2, "0");
    helpers.setValue(serialize({ date: `${y}-${m}-${d}`, hour, minute }));
    setOpen(false);
  };

  const selected = parsed;
  const isToday = (day: number) =>
    viewYear === now.getFullYear() &&
    viewMonth === now.getMonth() &&
    day === now.getDate();
  const isSelected = (day: number) =>
    selected !== null &&
    viewYear === new Date(`${selected.date}T00:00:00`).getFullYear() &&
    viewMonth === new Date(`${selected.date}T00:00:00`).getMonth() &&
    day === new Date(`${selected.date}T00:00:00`).getDate();

  const commitTime = (h: number, m: number) => {
    const date =
      parsed?.date ??
      `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
    helpers.setValue(serialize({ date, hour: h, minute: m }));
  };

  const adjustHour = (delta: number) =>
    setHour((h) => {
      const next = (h + delta + 24) % 24;
      commitTime(next, minute);
      return next;
    });

  const adjustMinute = (delta: number) =>
    setMinute((m) => {
      const next = (m + delta + 60) % 60;
      commitTime(hour, next);
      return next;
    });

  const hourItems = hour12
    ? Array.from({ length: 12 }, (_, i) => i + 1)
    : Array.from({ length: 24 }, (_, i) => i);
  const minuteItems = Array.from({ length: 12 }, (_, i) => i * 5);

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
            {parsed
              ? formatDisplay(parsed, locale, hour12)
              : t("common.selectDateTime")}
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
                      ? () => setViewYear((y) => y - 10)
                      : view === "months"
                        ? () => setViewYear((y) => y - 1)
                        : prevMonth
                  }
                  className="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
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
                      ? () => setViewYear((y) => y + 10)
                      : view === "months"
                        ? () => setViewYear((y) => y + 1)
                        : nextMonth
                  }
                  className="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
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

              {/* Days */}
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
                        className={classNames(
                          "rounded-lg py-1.5 text-xs transition-all duration-200",
                          isSelected(day) &&
                            "bg-primary-500 font-medium text-white",
                          !isSelected(day) &&
                            isToday(day) &&
                            "border border-primary-400 font-medium text-primary-600 bg-primary-50 dark:bg-primary-900/15 dark:text-primary-400",
                          !(isSelected(day) || isToday(day)) &&
                            "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800",
                        )}
                      >
                        {day}
                      </button>
                    );
                  })}
                </div>
              )}

              {/* Months */}
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

              {/* Years */}
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

              {/* Time picker */}
              <div className="relative mt-3 flex items-center justify-center gap-2 border-t border-gray-200 pt-3 dark:border-gray-700">
                {/* Hour */}
                <div className="flex flex-col items-center gap-1">
                  <button
                    type="button"
                    onClick={() => adjustHour(1)}
                    className="rounded p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  >
                    <IconChevronRight size={14} className="rotate-[-90deg]" />
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setTimeWheel(timeWheel === "hour" ? null : "hour")
                    }
                    className={classNames(
                      "w-8 rounded text-center text-sm font-medium tabular-nums transition",
                      timeWheel === "hour"
                        ? "bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300"
                        : "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800",
                    )}
                  >
                    {String(hour12 ? hour % 12 || 12 : hour).padStart(2, "0")}
                  </button>
                  <button
                    type="button"
                    onClick={() => adjustHour(-1)}
                    className="rounded p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  >
                    <IconChevronRight size={14} className="rotate-90" />
                  </button>
                </div>
                <span className="text-sm font-bold text-gray-400">:</span>
                {/* Minute */}
                <div className="flex flex-col items-center gap-1">
                  <button
                    type="button"
                    onClick={() => adjustMinute(1)}
                    className="rounded p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  >
                    <IconChevronRight size={14} className="rotate-[-90deg]" />
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setTimeWheel(timeWheel === "minute" ? null : "minute")
                    }
                    className={classNames(
                      "w-8 rounded text-center text-sm font-medium tabular-nums transition",
                      timeWheel === "minute"
                        ? "bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300"
                        : "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800",
                    )}
                  >
                    {String(minute).padStart(2, "0")}
                  </button>
                  <button
                    type="button"
                    onClick={() => adjustMinute(-1)}
                    className="rounded p-0.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  >
                    <IconChevronRight size={14} className="rotate-90" />
                  </button>
                </div>
                {hour12 && (
                  <button
                    type="button"
                    onClick={() => adjustHour(hour >= 12 ? -12 : 12)}
                    className="ml-1 rounded-md border border-gray-200 px-1.5 py-0.5 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-800"
                  >
                    {hour >= 12 ? "PM" : "AM"}
                  </button>
                )}

                {/* Time wheel dropdown */}
                {timeWheel && (
                  <div
                    ref={wheelRef}
                    className={classNames(
                      "absolute bottom-full left-1/2 z-10 mb-1 max-h-32 w-16 -translate-x-1/2 overflow-auto rounded-md border border-gray-200 bg-white py-1 shadow-lg",
                      "dark:border-gray-700 dark:bg-gray-900",
                    )}
                  >
                    {(timeWheel === "hour" ? hourItems : minuteItems).map(
                      (v) => {
                        const current = timeWheel === "hour" ? hour : minute;
                        const display = String(v).padStart(2, "0");
                        return (
                          <button
                            key={v}
                            type="button"
                            onClick={() => {
                              if (timeWheel === "hour") {
                                setHour(v);
                                commitTime(v, minute);
                              } else {
                                setMinute(v);
                                commitTime(hour, v);
                              }
                              setTimeWheel(null);
                            }}
                            className={classNames(
                              "block w-full px-2 py-1 text-center text-xs transition duration-100",
                              v === current &&
                                "bg-primary-500 font-medium text-white",
                              v !== current &&
                                "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800",
                            )}
                          >
                            {display}
                          </button>
                        );
                      },
                    )}
                  </div>
                )}
              </div>
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

export default DateTimePicker;
