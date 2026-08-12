import {
  IconStar,
  IconStarFilled,
  IconStarHalfFilled,
} from "@tabler/icons-react";
import classNames from "classnames";
import { type FC, useCallback, useState } from "react";

interface RatingProps {
  value: number;
  onChange?: (value: number) => void;
  count?: number;
  size?: "sm" | "md" | "lg";
  disabled?: boolean;
  allowHalf?: boolean;
  className?: string;
}

const sizeMap = { sm: 16, md: 20, lg: 24 };

const Rating: FC<RatingProps> = ({
  value,
  onChange,
  count = 5,
  size = "md",
  disabled = false,
  allowHalf = false,
  className,
}) => {
  const [hoverValue, setHoverValue] = useState<number | null>(null);
  const iconSize = sizeMap[size];

  const handleClick = useCallback(
    (i: number, half: boolean) => {
      if (disabled || !onChange) {
        return;
      }
      onChange(half && allowHalf ? i + 0.5 : i + 1);
    },
    [disabled, onChange, allowHalf],
  );

  const stars = [];
  for (let i = 0; i < count; i++) {
    const displayValue = hoverValue ?? value;
    const isFilled = i < Math.floor(displayValue);
    const isHalf = !isFilled && i < displayValue;

    stars.push(
      <div key={i} className="relative inline-flex">
        <button
          type="button"
          disabled={disabled}
          className="relative cursor-pointer bg-transparent p-0 transition-transform duration-200 hover:scale-110"
          onMouseEnter={() => setHoverValue(i + 1)}
          onMouseLeave={() => setHoverValue(null)}
          onClick={() => handleClick(i, false)}
        >
          {isFilled ? (
            <IconStarFilled
              size={iconSize}
              className="text-amber-400 drop-shadow-sm"
            />
          ) : isHalf ? (
            <IconStarHalfFilled
              size={iconSize}
              className="text-amber-400 drop-shadow-sm"
            />
          ) : (
            <IconStar
              size={iconSize}
              className="text-gray-300 transition-colors duration-200 hover:text-amber-300 dark:text-gray-600 dark:hover:text-amber-400"
            />
          )}
        </button>
        {allowHalf && (
          <button
            type="button"
            disabled={disabled}
            className="absolute inset-y-0 right-0 w-1/2 cursor-pointer bg-transparent p-0"
            onMouseEnter={() => setHoverValue(i + 0.5)}
            onMouseLeave={() => setHoverValue(null)}
            onClick={() => handleClick(i, true)}
          />
        )}
      </div>,
    );
  }

  return (
    <div className={classNames("inline-flex items-center gap-0.5", className)}>
      {stars}
    </div>
  );
};

export default Rating;
