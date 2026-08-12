import {
  IconChevronLeft,
  IconChevronRight,
  IconPointFilled,
} from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useCallback, useMemo, useState } from "react";

interface CalendarProps {
  value?: Date;
  onChange?: (date: Date) => void;
  minDate?: Date;
  maxDate?: Date;
  className?: string;
}

const MONTHS = [
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

const DAY_HEADERS = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"];

const isSameDay = (a: Date, b: Date): boolean =>
  a.getFullYear() === b.getFullYear() &&
  a.getMonth() === b.getMonth() &&
  a.getDate() === b.getDate();

const isInRange = (date: Date, min?: Date, max?: Date): boolean => {
  const d = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  if (min) {
    const m = new Date(min.getFullYear(), min.getMonth(), min.getDate());
    if (d < m) {
      return false;
    }
  }
  if (max) {
    const m = new Date(max.getFullYear(), max.getMonth(), max.getDate());
    if (d > m) {
      return false;
    }
  }
  return true;
};

const Calendar: FC<CalendarProps> = ({
  value,
  onChange,
  minDate,
  maxDate,
  className,
}) => {
  const today = useMemo(() => new Date(), []);

  const [viewDate, setViewDate] = useState(() => {
    if (value) {
      return new Date(value);
    }
    return new Date(today.getFullYear(), today.getMonth(), 1);
  });

  const viewYear = viewDate.getFullYear();
  const viewMonth = viewDate.getMonth();

  const weeks = useMemo(() => {
    const firstDay = new Date(viewYear, viewMonth, 1);
    const start = new Date(firstDay);
    const dayOfWeek = (start.getDay() + 6) % 7;
    start.setDate(start.getDate() - dayOfWeek);

    const msPerDay = 86400000;
    const result: Date[][] = [];
    let currentTs = start.getTime();

    for (let w = 0; w < 6; w++) {
      const week: Date[] = [];
      for (let d = 0; d < 7; d++) {
        week.push(new Date(currentTs));
        currentTs += msPerDay;
      }
      result.push(week);
      if (new Date(currentTs).getMonth() !== viewMonth && w >= 3) {
        break;
      }
    }

    return result;
  }, [viewYear, viewMonth]);

  const goPrevious = useCallback(() => {
    setViewDate((d) => new Date(d.getFullYear(), d.getMonth() - 1, 1));
  }, []);

  const goNext = useCallback(() => {
    setViewDate((d) => new Date(d.getFullYear(), d.getMonth() + 1, 1));
  }, []);

  const goToday = useCallback(() => {
    setViewDate(new Date(today.getFullYear(), today.getMonth(), 1));
    onChange?.(today);
  }, [today, onChange]);

  const handleSelect = useCallback(
    (date: Date) => {
      if (!isInRange(date, minDate, maxDate)) {
        return;
      }
      onChange?.(date);
    },
    [onChange, minDate, maxDate],
  );

  return (
    <div
      className={classNames(
        "select-none rounded-xl border border-gray-200/60 bg-gray-50 p-4 dark:border-gray-700/60 dark:bg-gray-800/50",
        className,
      )}
    >
      <div className="mb-3 flex items-center justify-between">
        <button
          type="button"
          onClick={goPrevious}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 transition-all duration-200 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200"
        >
          <IconChevronLeft size={18} />
        </button>
        <div className="flex items-center gap-2">
          <span className="text-base font-bold text-gray-900 dark:text-gray-50">
            {MONTHS[viewMonth]} {viewYear}
          </span>
          <button
            type="button"
            onClick={goToday}
            className="flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-600 transition-all duration-200 dark:bg-primary-900/25 dark:text-primary-400"
          >
            <IconPointFilled size={10} />
            Today
          </button>
        </div>
        <button
          type="button"
          onClick={goNext}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-500 transition-all duration-200 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200"
        >
          <IconChevronRight size={18} />
        </button>
      </div>

      <div className="grid grid-cols-7 gap-1">
        {DAY_HEADERS.map((d) => (
          <div
            key={d}
            className="mb-1 text-center text-xs font-semibold text-gray-400 dark:text-gray-500"
          >
            {d}
          </div>
        ))}

        {weeks.map((week, wi) => (
          <div key={wi} className="contents">
            {week.map((day, di) => {
              const isCurrentMonth = day.getMonth() === viewMonth;
              const isSelected = value ? isSameDay(day, value) : false;
              const isToday = isSameDay(day, today);
              const disabled = !isInRange(day, minDate, maxDate);

              return (
                <button
                  key={`${wi}-${di}`}
                  type="button"
                  onClick={() => handleSelect(day)}
                  disabled={disabled}
                  className={classNames(
                    "flex h-10 w-full items-center justify-center rounded-lg text-sm font-medium transition-all duration-200",
                    disabled
                      ? "cursor-not-allowed text-gray-300 dark:text-gray-700"
                      : isSelected
                        ? "bg-primary-500 text-white"
                        : "text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800",
                    !(isCurrentMonth || isSelected) &&
                      "text-gray-300 dark:text-gray-700",
                    isToday &&
                      !isSelected &&
                      "font-bold text-primary-600 dark:text-primary-400 underline decoration-primary-400/50 underline-offset-2",
                  )}
                >
                  {day.getDate()}
                </button>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
};

export default Calendar;
