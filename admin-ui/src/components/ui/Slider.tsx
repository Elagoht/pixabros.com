import classNames from "classnames";
import { type FC, useCallback, useEffect, useRef, useState } from "react";
import { clamp, roundStep } from "@/utilities/math";

interface SliderProps {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  disabled?: boolean;
  showValue?: boolean;
  className?: string;
  trackClassName?: string;
}

const Slider: FC<SliderProps> = ({
  value,
  onChange,
  min = 0,
  max = 100,
  step = 1,
  disabled = false,
  showValue = false,
  className,
  trackClassName,
}) => {
  const trackRef = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState(false);

  const getValueFromPosition = useCallback(
    (clientX: number) => {
      const rect = trackRef.current?.getBoundingClientRect();
      if (!rect) {
        return min;
      }
      const ratio = (clientX - rect.left) / rect.width;
      return roundStep(min + clamp(ratio, 0, 1) * (max - min), step);
    },
    [min, max, step],
  );

  const handlePointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (disabled) {
        return;
      }
      e.preventDefault();
      (e.target as HTMLElement).setPointerCapture(e.pointerId);
      onChange(getValueFromPosition(e.clientX));
      setDragging(true);
    },
    [disabled, getValueFromPosition, onChange],
  );

  const handlePointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (!dragging || disabled) {
        return;
      }
      onChange(getValueFromPosition(e.clientX));
    },
    [dragging, disabled, getValueFromPosition, onChange],
  );

  const handlePointerUp = useCallback(() => {
    setDragging(false);
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (disabled) {
        return;
      }
      let next = value;
      if (e.key === "ArrowRight" || e.key === "ArrowUp") {
        next = value + step;
      } else if (e.key === "ArrowLeft" || e.key === "ArrowDown") {
        next = value - step;
      } else if (e.key === "Home") {
        next = min;
      } else if (e.key === "End") {
        next = max;
      } else {
        return;
      }
      e.preventDefault();
      onChange(clamp(roundStep(next, step), min, max));
    },
    [disabled, value, step, min, max, onChange],
  );

  useEffect(() => {
    if (!dragging) {
      return;
    }
    const handler = () => setDragging(false);
    window.addEventListener("pointerup", handler);
    return () => window.removeEventListener("pointerup", handler);
  }, [dragging]);

  const pct = ((value - min) / (max - min)) * 100;

  return (
    <div className={classNames("flex items-center gap-3", className)}>
      <div
        ref={trackRef}
        role="slider"
        tabIndex={disabled ? -1 : 0}
        aria-valuemin={min}
        aria-valuemax={max}
        aria-valuenow={value}
        aria-disabled={disabled}
        className={classNames(
          "relative h-2 w-full cursor-pointer rounded-full transition-all duration-200",
          disabled ? "pointer-events-none opacity-50" : "",
          trackClassName ?? "bg-gray-200 dark:bg-gray-700",
        )}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onKeyDown={handleKeyDown}
      >
        <div
          className="absolute left-0 top-0 h-full rounded-full bg-primary-500"
          style={{ width: `${pct}%` }}
        />
        <div
          className={classNames(
            "absolute top-1/2 h-5 w-5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-primary-500 transition-all duration-200",
            dragging &&
              "ring-2 ring-primary-200 ring-offset-2 dark:ring-primary-800",
          )}
          style={{ left: `${pct}%` }}
        />
      </div>
      {showValue && (
        <span className="w-10 text-right text-sm font-medium tabular-nums text-gray-700 dark:text-gray-300">
          {value}
        </span>
      )}
    </div>
  );
};

export default Slider;
