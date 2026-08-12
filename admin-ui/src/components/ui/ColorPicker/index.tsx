import { IconPencil, IconX } from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useCallback, useEffect, useRef, useState } from "react";
import {
  hexToRgb,
  hsvToRgb,
  isValidHex,
  rgbToHex,
  rgbToHsv,
} from "@/utilities/color";
import { clamp } from "@/utilities/math";
import ColorSlider from "./ColorSlider";

interface ColorPickerProps {
  value: string;
  onChange: (color: string) => void;
  className?: string;
}

const ColorPicker: FC<ColorPickerProps> = ({ value, onChange, className }) => {
  const [open, setOpen] = useState(false);
  const [hexInput, setHexInput] = useState(value.toUpperCase());

  const rgb = hexToRgb(value);
  const hsv = rgbToHsv(...rgb);
  const [h, s, v] = hsv;
  const [r, g, b] = rgb;

  useEffect(() => {
    setHexInput(value.toUpperCase());
  }, [value]);

  const updateFromHsv = useCallback(
    (nh: number, ns: number, nv: number) => {
      nh = clamp(nh, 0, 360);
      ns = clamp(ns, 0, 100);
      nv = clamp(nv, 0, 100);
      const [nr, ng, nb] = hsvToRgb(nh, ns, nv);
      const hex = rgbToHex(nr, ng, nb);
      setHexInput(hex.toUpperCase());
      onChange(hex);
    },
    [onChange],
  );

  const updateFromHex = useCallback(
    (hex: string) => {
      setHexInput(hex.toUpperCase());
      if (isValidHex(hex)) {
        onChange(hex);
      }
    },
    [onChange],
  );

  const wrapperRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handler = (e: MouseEvent) => {
      if (
        wrapperRef.current &&
        !wrapperRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const thumbColor = rgbToHex(r, g, b);

  return (
    <div ref={wrapperRef} className={classNames("relative", className)}>
      <div className="flex items-center gap-2">
        <div
          className="h-8 w-8 shrink-0 rounded-md border border-gray-200 dark:border-gray-700"
          style={{ background: value }}
        />
        <input
          type="text"
          value={hexInput}
          onChange={(e) => updateFromHex(e.target.value)}
          className={classNames(
            "flex-1 rounded-md border px-3 py-1.5 font-mono text-sm outline-none transition",
            isValidHex(hexInput)
              ? "border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800/50"
              : "border-red-300 bg-gray-50 dark:border-red-800 dark:bg-red-900/10",
          )}
        />
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className={classNames(
            "flex h-8 w-8 items-center justify-center rounded-md border transition",
            open
              ? "border-primary-400 text-primary-600 dark:border-primary-500 dark:text-primary-400"
              : "border-gray-200 text-gray-500 hover:border-gray-300 hover:text-gray-700 dark:border-gray-700 dark:text-gray-400 dark:hover:border-gray-600 dark:hover:text-gray-300",
          )}
        >
          {open ? <IconX size={16} /> : <IconPencil size={16} />}
        </button>
      </div>

      <div
        className={classNames(
          "absolute z-50 mt-2 w-64 rounded-lg border border-gray-200 bg-white p-4 shadow-lg dark:border-gray-700 dark:bg-gray-900",
          "origin-top transition-all duration-150",
          open
            ? "scale-100 opacity-100"
            : "pointer-events-none scale-95 opacity-0",
        )}
      >
        <div className="space-y-3">
          <ColorSlider
            label="H"
            value={h}
            min={0}
            max={360}
            thumbColor={thumbColor}
            trackStyle={{
              background:
                "linear-gradient(to right, #f00, #ff0, #0f0, #0ff, #00f, #f0f, #f00)",
            }}
            onChange={(newH) => updateFromHsv(newH, s, v)}
          />
          <ColorSlider
            label="S"
            value={s}
            min={0}
            max={100}
            thumbColor={thumbColor}
            trackStyle={{
              background: `linear-gradient(to right, hsl(${h},0%,${v}%), hsl(${h},100%,${v}%))`,
            }}
            onChange={(newS) => updateFromHsv(h, newS, v)}
          />
          <ColorSlider
            label="V"
            value={v}
            min={0}
            max={100}
            thumbColor={thumbColor}
            trackStyle={{
              background: `linear-gradient(to right, hsl(${h},${s}%,0%), hsl(${h},${s}%,100%))`,
            }}
            onChange={(newV) => updateFromHsv(h, s, newV)}
          />
        </div>
        <div className="pointer-events-none absolute -top-1 left-6 h-2 w-2 rotate-45 border-l border-t border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900" />
      </div>
    </div>
  );
};

export default ColorPicker;
