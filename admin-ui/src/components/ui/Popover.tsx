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

interface PopoverProps {
  className?: string;
  children: ReactNode;
}

interface PopoverTriggerProps {
  className?: string;
  children: ReactNode;
}

interface PopoverContentProps {
  align?: "start" | "center" | "end";
  side?: "top" | "bottom";
  className?: string;
  children: ReactNode;
}

const Popover: FC<PopoverProps> & {
  Trigger: FC<PopoverTriggerProps>;
  Content: FC<PopoverContentProps>;
} = ({ className, children }) => {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState({ x: 0, y: 0, width: 0, height: 0 });
  const triggerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handler = (e: MouseEvent) => {
      if (
        contentRef.current &&
        !contentRef.current.contains(e.target as Node) &&
        triggerRef.current &&
        !triggerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open]);

  useEffect(() => {
    if (!open) {
      return;
    }
    const handler = () => setOpen(false);
    window.addEventListener("scroll", handler, true);
    return () => window.removeEventListener("scroll", handler, true);
  }, [open]);

  const toggle = useCallback(() => {
    if (!open) {
      const rect = triggerRef.current?.getBoundingClientRect();
      if (rect) {
        setPos({
          x: rect.left,
          y: rect.top,
          width: rect.width,
          height: rect.height,
        });
      }
    }
    setOpen((v) => !v);
  }, [open]);

  const childArray = Array.isArray(children) ? children : [children];

  const trigger = childArray.find(
    (c) =>
      typeof c === "object" &&
      "type" in c &&
      (c.type as FC).displayName === "PopoverTrigger",
  ) as React.ReactElement<PopoverTriggerProps> | undefined;

  const content = childArray.find(
    (c) =>
      typeof c === "object" &&
      "type" in c &&
      (c.type as FC).displayName === "PopoverContent",
  ) as React.ReactElement<PopoverContentProps> | undefined;

  const align = content?.props.align ?? "center";
  const side = content?.props.side ?? "bottom";

  return (
    <>
      <div
        ref={triggerRef}
        className={classNames("inline-flex", className)}
        onClick={toggle}
      >
        {trigger?.props.children}
      </div>
      {content &&
        createPortal(
          <div
            ref={contentRef}
            className={classNames(
              "fixed z-50 min-w-[16rem] rounded-md border border-gray-200 bg-white p-4 shadow-lg dark:border-gray-700 dark:bg-gray-900",
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
                    ? pos.x + pos.width - 256
                    : pos.x + pos.width / 2 - 128,
              top: side === "bottom" ? pos.y + pos.height + 8 : pos.y - 8 - 200,
            }}
          >
            {content.props.children}
          </div>,
          document.body,
        )}
    </>
  );
};

const PopoverTrigger: FC<PopoverTriggerProps> = ({ children }) => (
  <>{children}</>
);
PopoverTrigger.displayName = "PopoverTrigger";

const PopoverContent: FC<PopoverContentProps> = () => null;
PopoverContent.displayName = "PopoverContent";

Popover.Trigger = PopoverTrigger;
Popover.Content = PopoverContent;

export default Popover;
