import { type FC, useCallback, useEffect, useRef, useState } from "react";
import { clamp } from "@/utilities/math";

interface ColorSliderProps {
  label: string;
  value: number;
  min: number;
  max: number;
  thumbColor: string;
  trackStyle: React.CSSProperties;
  onChange: (value: number) => void;
}

const ColorSlider: FC<ColorSliderProps> = ({
  label,
  value,
  min,
  max,
  thumbColor,
  trackStyle,
  onChange,
}) => {
  const trackRef = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState(false);

  const compute = (clientX: number) => {
    const rect = trackRef.current?.getBoundingClientRect();
    if (!rect) {
      return value;
    }
    const pct = (clientX - rect.left) / rect.width;
    return clamp(pct * (max - min) + min, min, max);
  };

  const handleDown = (e: React.PointerEvent) => {
    e.preventDefault();
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
    setDragging(true);
    onChange(compute(e.clientX));
  };

  const handleMove = (e: React.PointerEvent) => {
    if (!dragging) {
      return;
    }
    onChange(compute(e.clientX));
  };

  const handleUp = useCallback(() => setDragging(false), []);

  useEffect(() => {
    if (!dragging) {
      return;
    }

    window.addEventListener("pointerup", handleUp);

    return () => window.removeEventListener("pointerup", handleUp);
  }, [dragging, handleUp]);

  const pct = ((value - min) / (max - min)) * 100;

  return (
    <div className="flex items-center gap-2">
      <span className="w-4 text-xs font-medium text-gray-500 dark:text-gray-400">
        {label}
      </span>
      <div
        ref={trackRef}
        className="relative flex-1"
        onPointerDown={handleDown}
        onPointerMove={handleMove}
      >
        <div className="h-2 cursor-pointer rounded-full" style={trackStyle} />
        <div
          className="pointer-events-none absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white shadow"
          style={{ left: `${pct}%`, background: thumbColor }}
        />
      </div>
      <span className="w-8 text-right text-xs tabular-nums text-gray-700 dark:text-gray-300">
        {Math.round(value)}
      </span>
    </div>
  );
};

export default ColorSlider;
