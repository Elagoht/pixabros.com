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

interface HoverCardProps {
  openDelay?: number;
  closeDelay?: number;
  className?: string;
  children: ReactNode;
}

interface HoverCardTriggerProps {
  className?: string;
  children: ReactNode;
}

interface HoverCardContentProps {
  align?: "start" | "center" | "end";
  side?: "top" | "bottom";
  className?: string;
  children: ReactNode;
}

const HoverCard: FC<HoverCardProps> & {
  Trigger: FC<HoverCardTriggerProps>;
  Content: FC<HoverCardContentProps>;
} = ({ openDelay = 300, closeDelay = 200, className, children }) => {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0, width: 0, height: 0 });
  const triggerRef = useRef<HTMLDivElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const openTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const closeTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handler = () => setOpen(false);
    window.addEventListener("scroll", handler, true);
    return () => window.removeEventListener("scroll", handler, true);
  }, [open]);

  // Cleanup timers on unmount
  useEffect(() => {
    return () => {
      clearTimeout(openTimer.current);
      clearTimeout(closeTimer.current);
    };
  }, []);

  const scheduleOpen = useCallback(() => {
    clearTimeout(closeTimer.current);
    openTimer.current = setTimeout(() => {
      const rect = triggerRef.current?.getBoundingClientRect();
      if (rect) {
        setPos({
          x: rect.left,
          y: rect.top,
          width: rect.width,
          height: rect.height,
        });
        setOpen(true);
      }
    }, openDelay);
  }, [openDelay]);

  const scheduleClose = useCallback(() => {
    clearTimeout(openTimer.current);
    closeTimer.current = setTimeout(() => setOpen(false), closeDelay);
  }, [closeDelay]);

  const handleTriggerEnter = useCallback(() => {
    scheduleOpen();
  }, [scheduleOpen]);

  const handleTriggerLeave = useCallback(() => {
    scheduleClose();
  }, [scheduleClose]);

  const handleCardEnter = useCallback(() => {
    clearTimeout(closeTimer.current);
  }, []);

  const handleCardLeave = useCallback(() => {
    scheduleClose();
  }, [scheduleClose]);

  const childArray = children;

  const trigger = (() => {
    const arr = Array.isArray(childArray) ? childArray : [childArray];
    return arr.find(
      (c) =>
        typeof c === "object" &&
        "type" in c &&
        (c.type as FC).displayName === "HoverCardTrigger",
    ) as React.ReactElement<HoverCardTriggerProps> | undefined;
  })();

  const content = (() => {
    const arr = Array.isArray(childArray) ? childArray : [childArray];
    return arr.find(
      (c) =>
        typeof c === "object" &&
        "type" in c &&
        (c.type as FC).displayName === "HoverCardContent",
    ) as React.ReactElement<HoverCardContentProps> | undefined;
  })();

  const align = content?.props.align ?? "center";
  const side = content?.props.side ?? "bottom";

  return (
    <>
      <div
        ref={triggerRef}
        className={classNames("inline-flex", className)}
        onMouseEnter={handleTriggerEnter}
        onMouseLeave={handleTriggerLeave}
      >
        {trigger?.props.children}
      </div>
      {content &&
        createPortal(
          <div
            ref={cardRef}
            className={classNames(
              "fixed z-50 w-72 rounded-md border border-gray-200 bg-white p-4 shadow-lg dark:border-gray-700 dark:bg-gray-900",
              align === "start"
                ? "origin-top-left"
                : align === "end"
                  ? "origin-top-right"
                  : "origin-top",
              "transition-opacity duration-150",
              open ? "opacity-100" : "pointer-events-none opacity-0",
              content.props.className,
            )}
            style={{
              left:
                align === "start"
                  ? pos.x
                  : align === "end"
                    ? pos.x + pos.width - 288
                    : pos.x + pos.width / 2 - 144,
              top: side === "bottom" ? pos.y + pos.height + 8 : pos.y - 8 - 200,
            }}
            onMouseEnter={handleCardEnter}
            onMouseLeave={handleCardLeave}
          >
            {content.props.children}
          </div>,
          document.body,
        )}
    </>
  );
};

const HoverCardTrigger: FC<HoverCardTriggerProps> = ({ children }) => (
  <>{children}</>
);
HoverCardTrigger.displayName = "HoverCardTrigger";

const HoverCardContent: FC<HoverCardContentProps> = () => null;
HoverCardContent.displayName = "HoverCardContent";

HoverCard.Trigger = HoverCardTrigger;
HoverCard.Content = HoverCardContent;

export default HoverCard;
