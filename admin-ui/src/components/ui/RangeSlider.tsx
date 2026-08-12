import classNames from "classnames";
import { type FC, useCallback, useEffect, useRef, useState } from "react";
import { clamp, roundStep } from "@/utilities/math";

interface RangeSliderProps {
  minValue: number;
  maxValue: number;
  onChange: (min: number, max: number) => void;
  min?: number;
  max?: number;
  step?: number;
  disabled?: boolean;
  showValue?: boolean;
  className?: string;
  trackClassName?: string;
}

const RangeSlider: FC<RangeSliderProps> = ({
  minValue,
  maxValue,
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
  const [dragging, setDragging] = useState<"min" | "max" | null>(null);

  const getValueFromPosition = useCallback(
    (clientX: number) => {
      const rect = trackRef.current?.getBoundingClientRect();
      if (!rect) {
        return min;
      }
      const ratio = clamp((clientX - rect.left) / rect.width, 0, 1);
      return roundStep(min + ratio * (max - min), step);
    },
    [min, max, step],
  );

  const handlePointerDown = useCallback(
    (e: React.PointerEvent, thumb: "min" | "max") => {
      if (disabled) {
        return;
      }
      e.preventDefault();
      e.stopPropagation();
      (e.target as HTMLElement).setPointerCapture(e.pointerId);
      setDragging(thumb);
    },
    [disabled],
  );

  const handleTrackPointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (disabled) {
        return;
      }
      const val = getValueFromPosition(e.clientX);
      const distMin = Math.abs(val - minValue);
      const distMax = Math.abs(val - maxValue);
      const target = distMin <= distMax ? "min" : "max";
      const newMin = target === "min" ? Math.min(val, maxValue) : minValue;
      const newMax = target === "max" ? Math.max(val, minValue) : maxValue;
      onChange(newMin, newMax);
      setDragging(target);
    },
    [disabled, getValueFromPosition, minValue, maxValue, onChange],
  );

  useEffect(() => {
    if (!dragging) {
      return;
    }
    const handleMove = (e: PointerEvent) => {
      const val = getValueFromPosition(e.clientX);
      if (dragging === "min") {
        onChange(Math.min(val, maxValue), maxValue);
      } else {
        onChange(minValue, Math.max(val, minValue));
      }
    };
    const handleUp = () => setDragging(null);
    window.addEventListener("pointermove", handleMove);
    window.addEventListener("pointerup", handleUp);
    return () => {
      window.removeEventListener("pointermove", handleMove);
      window.removeEventListener("pointerup", handleUp);
    };
  }, [dragging, minValue, maxValue, onChange, getValueFromPosition]);

  const minPct = ((minValue - min) / (max - min)) * 100;
  const maxPct = ((maxValue - min) / (max - min)) * 100;

  return (
    <div className={classNames("flex items-center gap-3", className)}>
      <div
        ref={trackRef}
        className={classNames(
          "relative h-1.5 w-full cursor-pointer rounded-full transition-colors",
          disabled ? "pointer-events-none opacity-50" : "",
          trackClassName ?? "bg-gray-200 dark:bg-gray-700",
        )}
        onPointerDown={handleTrackPointerDown}
      >
        <div
          className="absolute top-0 h-full rounded-full bg-primary-400"
          style={{ left: `${minPct}%`, width: `${maxPct - minPct}%` }}
        />
        <div
          role="slider"
          tabIndex={disabled ? -1 : 0}
          aria-valuemin={min}
          aria-valuemax={maxValue}
          aria-valuenow={minValue}
          className={classNames(
            "absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-primary-400 shadow transition-shadow",
            dragging === "min" &&
              "ring-2 ring-primary-200 dark:ring-primary-800",
          )}
          style={{ left: `${minPct}%` }}
          onPointerDown={(e) => handlePointerDown(e, "min")}
        />
        <div
          role="slider"
          tabIndex={disabled ? -1 : 0}
          aria-valuemin={minValue}
          aria-valuemax={max}
          aria-valuenow={maxValue}
          className={classNames(
            "absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-primary-400 shadow transition-shadow",
            dragging === "max" &&
              "ring-2 ring-primary-200 dark:ring-primary-800",
          )}
          style={{ left: `${maxPct}%` }}
          onPointerDown={(e) => handlePointerDown(e, "max")}
        />
      </div>
      {showValue && (
        <span className="w-20 text-right text-sm font-medium tabular-nums text-gray-700 dark:text-gray-300">
          {minValue} - {maxValue}
        </span>
      )}
    </div>
  );
};

export default RangeSlider;
