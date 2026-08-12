import classNames from "classnames";
import {
  type FC,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

type TooltipPosition = "top" | "bottom" | "left" | "right";

interface TooltipProps {
  content: string;
  children: ReactNode;
  position?: TooltipPosition;
  className?: string;
  block?: boolean;
}

const getOffset = (rect: DOMRect, position: TooltipPosition) => {
  switch (position) {
    case "top":
      return { top: rect.top - 8, left: rect.left + rect.width / 2 };
    case "bottom":
      return { top: rect.bottom + 8, left: rect.left + rect.width / 2 };
    case "left":
      return { top: rect.top + rect.height / 2, left: rect.left - 8 };
    case "right":
      return { top: rect.top + rect.height / 2, left: rect.right + 8 };
  }
};

const transformClass: Record<TooltipPosition, string> = {
  top: "-translate-x-1/2 -translate-y-full",
  bottom: "-translate-x-1/2",
  left: "-translate-x-full -translate-y-1/2",
  right: "-translate-y-1/2",
};

const arrowClasses: Record<TooltipPosition, string> = {
  top: "top-full left-1/2 -translate-x-1/2 border-t-gray-900 border-x-transparent border-b-transparent dark:border-t-gray-700",
  bottom:
    "bottom-full left-1/2 -translate-x-1/2 border-b-gray-900 border-x-transparent border-t-transparent dark:border-b-gray-700",
  left: "left-full top-1/2 -translate-y-1/2 border-l-gray-900 border-y-transparent border-r-transparent dark:border-l-gray-700",
  right:
    "right-full top-1/2 -translate-y-1/2 border-r-gray-900 border-y-transparent border-l-transparent dark:border-r-gray-700",
};

const Tooltip: FC<TooltipProps> = ({
  content,
  children,
  position = "top",
  className,
  block,
}) => {
  const [visible, setVisible] = useState(false);
  const [coords, setCoords] = useState({ top: -9999, left: -9999 });
  const triggerRef = useRef<HTMLDivElement>(null);

  const update = useCallback(() => {
    if (!triggerRef.current) {
      return;
    }
    const rect = triggerRef.current.getBoundingClientRect();
    setCoords(getOffset(rect, position));
  }, [position]);

  useEffect(() => {
    if (visible) {
      update();
    } else {
      setCoords({ top: -9999, left: -9999 });
    }
  }, [visible, update]);

  return (
    <>
      <div
        ref={triggerRef}
        className={classNames(
          "relative",
          block ? "flex w-full" : "inline-flex",
          className,
        )}
        onMouseEnter={() => setVisible(true)}
        onMouseLeave={() => setVisible(false)}
      >
        {children}
      </div>
      {visible &&
        createPortal(
          <div
            className={classNames(
              "pointer-events-none fixed z-[9999] whitespace-nowrap rounded-lg px-3 py-1.5 text-xs font-medium text-white shadow-lg shadow-gray-500/30 transition-opacity duration-150",
              "bg-gray-800 dark:bg-gray-700",
              transformClass[position],
            )}
            style={{ top: coords.top, left: coords.left }}
          >
            {content}
            <span
              className={classNames(
                "absolute border-4",
                arrowClasses[position],
              )}
            />
          </div>,
          document.body,
        )}
    </>
  );
};

export default Tooltip;
